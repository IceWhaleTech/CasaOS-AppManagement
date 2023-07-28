package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

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

	// Global is a map to inject environment variables to the app.
	Global = make(map[string]string)

	CasaOSGlobalVariables = &model.CasaOSGlobalVariables{}

	Cfg               *ini.File
	ConfigFilePath    string
	GlobalEnvFilePath string
)

func InitSetup(config string, sample string) {
	ConfigFilePath = AppManagementConfigFilePath
	if len(config) > 0 {
		ConfigFilePath = config
	}

	if _, err := os.Stat(ConfigFilePath); os.IsNotExist(err) {
		fmt.Println("config file not exist, create it")
		// create config file
		file, err := os.Create(ConfigFilePath)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		// write default config
		_, err = file.WriteString(sample)
		if err != nil {
			panic(err)
		}
	}

	var err error

	Cfg, err = ini.LoadSources(ini.LoadOptions{Insensitive: true, AllowShadows: true}, ConfigFilePath)
	if err != nil {
		panic(err)
	}

	mapTo("common", CommonInfo)
	mapTo("app", AppInfo)
	mapTo("server", ServerInfo)
}

func SaveSetup() error {
	reflectFrom("common", CommonInfo)
	reflectFrom("app", AppInfo)
	reflectFrom("server", ServerInfo)

	return Cfg.SaveTo(ConfigFilePath)
}

func InitGlobal(config string) {
	// read file
	// file content like this:
	// OPENAI_API_KEY=123456

	// read file
	GlobalEnvFilePath = AppManagementGlobalEnvFilePath
	if len(config) > 0 {
		ConfigFilePath = config
	}

	// from file read key and value
	// set to Global
	file, err := os.Open(GlobalEnvFilePath)
	// there can't to panic err. because the env file is a new file
	// very much user didn't have the file.
	if err != nil {
		// log.Fatal will exit the program. So we only can to log the error.
		log.Println("open global env file error:", err)
	} else {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, "=")
			Global[parts[0]] = parts[1]
		}
	}
}

func SaveGlobal() error {
	// file content like this:
	// OPENAI_API_KEY=123456
	file, err := os.Create(AppManagementGlobalEnvFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for key, value := range Global {
		fmt.Fprintf(writer, "%s=%s\n", key, value)
	}

	writer.Flush()
	return err
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
