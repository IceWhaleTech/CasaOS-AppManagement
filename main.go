package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/sqlite"
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

	sqliteDB := sqlite.GetDb(*dbFlag)

	service.MyService = service.NewService(sqliteDB, config.CommonInfo.RuntimePath)

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
