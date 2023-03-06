package main

import (
	"flag"
	"fmt"
	"os"
	"reflect"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils"

	_logger "github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"gopkg.in/yaml.v3"
)

var (
	logger *Logger

	commit = "private build"
	date   = "private build"
)

func main() {
	versionFlag := flag.Bool("v", false, "version")

	inputFlag := flag.String("i", "", "input file")
	outputFlag := flag.String("o", "", "output file")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("v%s\n", common.AppManagementVersion)
		os.Exit(0)
	}

	println("git commit:", commit)
	println("build date:", date)

	if *inputFlag == "" {
		inputFlag = utils.Ptr("appfile.json")
	}

	if *outputFlag == "" {
		outputFlag = utils.Ptr(common.ComposeYAMLFileName)
	}

	logger = NewLogger()
	logger.Info("input file: %s", *inputFlag)
	logger.Info("output file: %s", *outputFlag)

	_logger.LogInitConsoleOnly()

	appFile, err := NewAppFile(*inputFlag)
	if err != nil {
		logger.Error("failed to load appfile: %s", err)
		os.Exit(1)
	}

	composeApp := appFile.ComposeApp()

	composeYAML, err := yaml.Marshal(composeApp)
	if err != nil {
		logger.Error("failed to marshal compose app converted from appfile: %s", err)
		os.Exit(1)
	}

	composeAppLoopBack, err := service.NewComposeAppFromYAML(composeYAML)
	if err != nil {
		logger.Error("failed to load compose app YAML converted from appfile: %s", err)
		os.Exit(1)
	}

	if err := Compare(composeApp, composeAppLoopBack); err != nil {
		logger.Error("failed to validate compose app YAML converted from appfile: %s", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputFlag, composeYAML, 0o600); err != nil {
		logger.Error("failed to write docker-compose.yml: %s", err)
		os.Exit(1)
	}
}

func Compare(composeApp1, composeApp2 *service.ComposeApp) error {
	storeInfo1, err := composeApp1.StoreInfo(true)
	if err != nil {
		return err
	}

	storeInfo2, err := composeApp2.StoreInfo(true)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(storeInfo1, storeInfo2) {
		return fmt.Errorf("store info of two compose apps does not deep equal")
	}

	mainApp1 := composeApp1.App(*storeInfo1.MainApp)
	mainApp2 := composeApp2.App(*storeInfo2.MainApp)

	mainAppStoreInfo1, err := mainApp1.StoreInfo()
	if err != nil {
		return err
	}

	mainAppStoreInfo2, err := mainApp2.StoreInfo()
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(mainAppStoreInfo1, mainAppStoreInfo2) {
		return fmt.Errorf("store info of two main apps does not deep equal")
	}

	return nil
}
