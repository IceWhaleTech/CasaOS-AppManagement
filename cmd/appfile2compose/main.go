package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	v2 "github.com/IceWhaleTech/CasaOS-AppManagement/service/v2"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gopkg.in/yaml.v3"
	"gotest.tools/v3/assert/cmp"
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
		outputFlag = utils.Ptr("docker-compose.yml")
	}

	logger = NewLogger()
	logger.Info("input file: %s", *inputFlag)
	logger.Info("output file: %s", *outputFlag)

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

	composeAppLoopBack, err := v2.LoadComposeApp(composeYAML)
	if err != nil {
		logger.Error("failed to load compose app YAML converted from appfile: %s", err)
		os.Exit(1)
	}

	if err := validateComposeApp(composeApp, composeAppLoopBack); err != nil {
		logger.Error("failed to validate compose app YAML converted from appfile: %s", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputFlag, composeYAML, 0o600); err != nil {
		logger.Error("failed to write docker-compose.yml: %s", err)
		os.Exit(1)
	}
}

func validateComposeApp(composeApp1, composeApp2 *v2.ComposeApp) error {
	storeInfo1, err := composeApp1.StoreInfo()
	if err != nil {
		return err
	}

	storeInfo2, err := composeApp2.StoreInfo()
	if err != nil {
		return err
	}

	if result := cmp.DeepEqual(storeInfo1, storeInfo2)(); !result.Success() {
		return fmt.Errorf("store info is not equal: %s", result)
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

	if result := cmp.DeepEqual(mainAppStoreInfo1, mainAppStoreInfo2, cmpopts.EquateEmpty())(); !result.Success() {
		return fmt.Errorf("main app is not equal: %s", result)
	}

	return nil
}
