package config

import (
	"log"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"gopkg.in/ini.v1"
)

var (
	CommonInfo = &model.CommonModel{
		RuntimePath: "/var/run/casaos",
	}

	AppInfo = &model.APPModel{
		DBPath:       "/var/lib/casaos",
		AppStorePath: "/var/lib/casaos/appstore",
		AppsPath:     "/var/lib/casaos/apps",
		LogPath:      "/var/log/casaos",
		LogSaveName:  common.AppManagementServiceName,
		LogFileExt:   "log",
	}

	ServerInfo = &model.ServerModel{
		AppStoreList: []string{},
	}

	Global = &model.GlobalModel{
		OpenAIAPIKey: "sk-123456",
	}

	CasaOSGlobalVariables = &model.CasaOSGlobalVariables{}

	Cfg            *ini.File
	ConfigFilePath string
)

func InitSetup(config string) {
	ConfigFilePath = AppManagementConfigFilePath
	if len(config) > 0 {
		ConfigFilePath = config
	}

	var err error

	Cfg, err = ini.LoadSources(ini.LoadOptions{Insensitive: true, AllowShadows: true}, ConfigFilePath)
	if err != nil {
		panic(err)
	}

	mapTo("common", CommonInfo)
	mapTo("app", AppInfo)
	mapTo("server", ServerInfo)
	mapTo("global", Global)
}

func SaveSetup() error {
	reflectFrom("common", CommonInfo)
	reflectFrom("app", AppInfo)
	reflectFrom("server", ServerInfo)

	return Cfg.SaveTo(ConfigFilePath)
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
