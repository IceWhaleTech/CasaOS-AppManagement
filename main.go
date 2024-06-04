//go:generate bash -c "mkdir -p codegen && go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.12.4 -generate types,server,spec -package codegen api/app_management/openapi.yaml > codegen/app_management_api.go"
//go:generate bash -c "mkdir -p codegen/message_bus && go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.12.4 -generate types,client -package message_bus https://raw.githubusercontent.com/IceWhaleTech/CasaOS-MessageBus/main/api/message_bus/openapi.yaml > codegen/message_bus/api.go"

package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-AppManagement/route"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/coreos/go-systemd/daemon"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	util_http "github.com/IceWhaleTech/CasaOS-Common/utils/http"
)

var (
	commit = "private build"
	date   = "private build"

	//go:embed api/index.html
	_docHTML string

	//go:embed api/index_v1.html
	_docHTMLV1 string

	//go:embed api/app_management/openapi.yaml
	_docYAML string

	//go:embed api/app_management/openapi_v1.yaml
	_docYAMLV1 string

	//go:embed build/sysroot/etc/casaos/app-management.conf.sample
	_confSample string
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// parse arguments and intialize
	{
		configFlag := flag.String("c", "", "config file path")
		versionFlag := flag.Bool("v", false, "version")
		removeRuntimeIfNoNvidiaGPUFlag := flag.Bool("removeRuntimeIfNoNvidiaGPU", false, "remove runtime with nvidia gpu")

		flag.Parse()

		if *versionFlag {
			fmt.Printf("v%s\n", common.AppManagementVersion)
			os.Exit(0)
		}

		println("git commit:", commit)
		println("build date:", date)

		config.InitSetup(*configFlag, _confSample)
		config.InitGlobal(*configFlag)

		logger.LogInit(config.AppInfo.LogPath, config.AppInfo.LogSaveName, config.AppInfo.LogFileExt)

		service.MyService = service.NewService(config.CommonInfo.RuntimePath)

		config.RemoveRuntimeIfNoNvidiaGPUFlag = *removeRuntimeIfNoNvidiaGPUFlag
	}

	// setup cron
	{
		crontab := cron.New(cron.WithSeconds())

		// schedule async v2job to get v2 appstore list
		go func() {
			// run once at startup
			if err := service.MyService.AppStoreManagement().UpdateCatalog(); err != nil {
				logger.Error("error when updating AppStore catalog at startup", zap.Error(err))
			}
		}()

		if _, err := crontab.AddFunc("@every 10m", func() {
			if err := service.MyService.AppStoreManagement().UpdateCatalog(); err != nil {
				logger.Error("error when updating AppStore catalog", zap.Error(err))
			}
		}); err != nil {
			panic(err)
		}

		crontab.Start()
		defer crontab.Stop()

	}

	// register at message bus
	{
		response, err := service.MyService.MessageBus().RegisterEventTypesWithResponse(ctx, common.EventTypes)
		if err != nil {
			logger.Error("error when trying to register one or more event types - some event type will not be discoverable", zap.Error(err))
		}

		if response != nil && response.StatusCode() != http.StatusOK {
			logger.Error("error when trying to register one or more event types - some event type will not be discoverable", zap.String("status", response.Status()), zap.String("body", string(response.Body)))
		}
	}

	// setup listener
	listener, err := net.Listen("tcp", net.JoinHostPort(common.Localhost, "0"))
	if err != nil {
		panic(err)
	}

	urlFilePath := filepath.Join(config.CommonInfo.RuntimePath, "app-management.url")
	if err := file.CreateFileAndWriteContent(urlFilePath, "http://"+listener.Addr().String()); err != nil {
		logger.Error("error when creating address file", zap.Error(err),
			zap.Any("address", listener.Addr().String()),
			zap.Any("filepath", urlFilePath),
		)
	}

	// initialize routers and register at gateway
	{
		apiPaths := []string{
			"/v1/apps",
			"/v1/container",
			"/v1/app-categories",
			route.V1DocPath,
			route.V2APIPath,
			route.V2DocPath,
		}

		for _, apiPath := range apiPaths {
			if err := service.MyService.Gateway().CreateRoute(&model.Route{
				Path:   apiPath,
				Target: "http://" + listener.Addr().String(),
			}); err != nil {
				panic(err)
			}
		}
	}

	v1Router := route.InitV1Router()
	v2Router := route.InitV2Router()
	v1DocRouter := route.InitV1DocRouter(_docHTMLV1, _docYAMLV1)
	v2DocRouter := route.InitV2DocRouter(_docHTML, _docYAML)

	mux := &util_http.HandlerMultiplexer{
		HandlerMap: map[string]http.Handler{
			"v1":    v1Router,
			"v2":    v2Router,
			"v1doc": v1DocRouter,
			"doc":   v2DocRouter,
		},
	}

	// notify systemd that we are ready
	{
		if supported, err := daemon.SdNotify(false, daemon.SdNotifyReady); err != nil {
			logger.Error("Failed to notify systemd that casaos main service is ready", zap.Any("error", err))
		} else if supported {
			logger.Info("Notified systemd that casaos main service is ready")
		} else {
			logger.Info("This process is not running as a systemd service.")
		}

		logger.Info("App management service is listening...", zap.Any("address", listener.Addr().String()))
	}

	s := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second, // fix G112: Potential slowloris attack (see https://github.com/securego/gosec)
	}

	err = s.Serve(listener) // not using http.serve() to fix G114: Use of net/http serve function that has no support for setting timeouts (see https://github.com/securego/gosec)
	if err != nil {
		panic(err)
	}
}
