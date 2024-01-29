package main

import (
	"fmt"
	"os"

	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	utils_logger "github.com/IceWhaleTech/CasaOS-Common/utils/logger"
)

var logger = NewLogger()

func main() {

	utils_logger.LogInit(config.AppInfo.LogPath, config.AppInfo.LogSaveName, config.AppInfo.LogFileExt)

	if len(os.Args) < 1 {
		os.Args = append(os.Args, "-h")
	}

	dockerComposeFilePath := os.Args[1]

	// check file exists
	if _, err := os.Stat(dockerComposeFilePath); os.IsNotExist(err) {
		logger.Error("docker-compose file does not exist: %s", dockerComposeFilePath)
		os.Exit(1)
	}

	composeFileContent := file.ReadFullFile(dockerComposeFilePath)

	_, err := service.NewComposeAppFromYAML(composeFileContent, false, false)
	if err != nil {
		logger.Error("failed to parse docker-compose file %s", err.Error())
		os.Exit(1)
	}
	fmt.Println("pass validate")
}
