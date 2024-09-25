package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/constants"
	"gopkg.in/ini.v1"
)

var (
	CommonInfo = &model.CommonModel{
		RuntimePath: constants.DefaultRuntimePath,
	}

	AppInfo = &model.APPModel{
		AppStorePath: filepath.Join(constants.DefaultDataPath, "appstore"),
		AppsPath:     filepath.Join(constants.DefaultDataPath, "apps"),
		LogPath:      constants.DefaultLogPath,
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

func ReloadConfig() {
	var err error
	Cfg, err = ini.LoadSources(ini.LoadOptions{Insensitive: true, AllowShadows: true}, ConfigFilePath)
	if err != nil {
		fmt.Println("failed to reload config", err)
	} else {
		mapTo("common", CommonInfo)
		mapTo("app", AppInfo)
		mapTo("server", ServerInfo)
	}
}

func InitSetup(config string, sample string) {
	ConfigFilePath = AppManagementConfigFilePath
	if len(config) > 0 {
		ConfigFilePath = config
	}

	// create default config file if not exist
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
