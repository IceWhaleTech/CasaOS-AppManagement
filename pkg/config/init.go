package config

import (
	"log"
	"os"
	"strings"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"gopkg.in/ini.v1"
)

const (
	EnvDisableAppStoreCache = "CASAOS_DISABLE_APP_STORE_CACHE"
)

var (
	CommonInfo = &model.CommonModel{
		RuntimePath: "/var/run/casaos",
	}

	AppInfo = &model.APPModel{
		DBPath:      "/var/lib/casaos",
		LogPath:     "/var/log/casaos",
		LogSaveName: common.AppManagementServiceName,
		LogFileExt:  "log",
	}

	ServerInfo = &model.ServerModel{
		ServerAPI: "https://api.casaos.io/casaos-api",
	}

	CasaOSGlobalVariables = &model.CasaOSGlobalVariables{}

	Cfg            *ini.File
	ConfigFilePath string

	DebugInfo = struct {
		DisableAppStoreCache bool
	}{
		DisableAppStoreCache: false,
	}
)

func InitSetup(config string) {
	// load from config file
	ConfigFilePath = AppManagementConfigFilePath
	if len(config) > 0 {
		ConfigFilePath = config
	}

	var err error

	Cfg, err = ini.Load(ConfigFilePath)
	if err != nil {
		panic(err)
	}

	mapTo("common", CommonInfo)
	mapTo("app", AppInfo)
	mapTo("server", ServerInfo)

	// load from environment variables
	if disableAppStoreCache, ok := os.LookupEnv(EnvDisableAppStoreCache); ok {
		DebugInfo.DisableAppStoreCache = strings.ToLower(disableAppStoreCache) == "true" || strings.ToLower(disableAppStoreCache) == "yes" || disableAppStoreCache == "1"
	}
}

func SaveSetup(config string) {
	reflectFrom("common", CommonInfo)
	reflectFrom("app", AppInfo)
	reflectFrom("server", ServerInfo)

	configFilePath := AppManagementConfigFilePath
	if len(config) > 0 {
		configFilePath = config
	}

	if err := Cfg.SaveTo(configFilePath); err != nil {
		log.Printf("error when saving to %s", configFilePath)
		panic(err)
	}
}

func mapTo(section string, v interface{}) {
	err := Cfg.Section(section).MapTo(v)
	if err != nil {
		log.Fatalf("Cfg.MapTo %s err: %v", section, err)
	}
}

func reflectFrom(section string, v interface{}) {
	err := Cfg.Section(section).ReflectFrom(v)
	if err != nil {
		log.Fatalf("Cfg.ReflectFrom %s err: %v", section, err)
	}
}
