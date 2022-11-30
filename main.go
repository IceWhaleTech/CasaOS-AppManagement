//go:generate bash -c "mkdir -p codegen/message_bus && go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.12.2 -generate types,client -package message_bus https://raw.githubusercontent.com/IceWhaleTech/CasaOS-MessageBus/main/api/message_bus/openapi.yaml > codegen/message_bus/api.go"

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen/message_bus"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-AppManagement/route"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/coreos/go-systemd/daemon"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

const localhost = "127.0.0.1"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// arguments
	configFlag := flag.String("c", "", "config file path")
	dbFlag := flag.String("db", "", "db path")
	versionFlag := flag.Bool("v", false, "version")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("v%s\n", common.AppManagementVersion)
		os.Exit(0)
	}

	config.InitSetup(*configFlag)

	logger.LogInit(config.AppInfo.LogPath, config.AppInfo.LogSaveName, config.AppInfo.LogFileExt)

	if len(*dbFlag) == 0 {
		*dbFlag = config.AppInfo.DBPath
	}

	service.MyService = service.NewService(config.CommonInfo.RuntimePath)

	service.Cache = cache.New(5*time.Minute, 60*time.Second)

	service.GetToken()

	service.GetToken()

	service.NewVersionApp = make(map[string]string)

	listener, err := net.Listen("tcp", net.JoinHostPort(localhost, "0"))
	if err != nil {
		panic(err)
	}

	// register at gateway
	for _, v := range []string{"apps", "container", "app-categories"} {
		if err := service.MyService.Gateway().CreateRoute(&model.Route{
			Path:   "/v1/" + v,
			Target: "http://" + listener.Addr().String(),
		}); err != nil {
			panic(err)
		}
	}

	// register at message bus
	for _, eventType := range []message_bus.EventType{
		common.EventTypeContainerAppInstalling,
		common.EventTypeContainerAppInstalled,
		common.EventTypeContainerAppInstallFailed,
		common.EventTypeContainerAppUninstalling,
		common.EventTypeContainerAppUninstalled,
		common.EventTypeContainerAppUninstallFailed,
	} {
		if _, err := service.MyService.MessageBus().RegisterEventTypeWithResponse(ctx, eventType); err != nil {
			logger.Error("error when trying to register event type - the event type will not be discoverable by subscribers", zap.Error(err), zap.Any("event type", eventType))
		}
	}

	r := route.InitRouter()

	s := &http.Server{
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second, // fix G112: Potential slowloris attack (see https://github.com/securego/gosec)
	}

	// notify systemd that we are ready
	if supported, err := daemon.SdNotify(false, daemon.SdNotifyReady); err != nil {
		logger.Error("Failed to notify systemd that casaos main service is ready", zap.Any("error", err))
	} else if supported {
		logger.Info("Notified systemd that casaos main service is ready")
	} else {
		logger.Info("This process is not running as a systemd service.")
	}

	logger.Info("App management service is listening...", zap.Any("address", listener.Addr().String()))

	err = s.Serve(listener) // not using http.serve() to fix G114: Use of net/http serve function that has no support for setting timeouts (see https://github.com/securego/gosec)
	if err != nil {
		panic(err)
	}
}
