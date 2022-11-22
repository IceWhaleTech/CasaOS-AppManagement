package config

import (
	"log"

	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"gopkg.in/ini.v1"
)

var (
	CommonInfo = &model.CommonModel{
		RuntimePath: "/var/run/casaos",
	}

	// TODO - add default values
	AppInfo               = &model.APPModel{}
	ServerInfo            = &model.ServerModel{}
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

	Cfg, err = ini.Load(ConfigFilePath)
	if err != nil {
		panic(err)
	}

	mapTo("common", CommonInfo)
	mapTo("app", AppInfo)
	mapTo("server", ServerInfo)
}

func mapTo(section string, v interface{}) {
	err := Cfg.Section(section).MapTo(v)
	if err != nil {
		log.Fatalf("Cfg.MapTo %s err: %v", section, err)
	}
}
