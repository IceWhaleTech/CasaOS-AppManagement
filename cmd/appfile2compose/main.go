package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"gopkg.in/yaml.v2"
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
		logger.Error("failed to marshal docker-compose.yml: %s", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputFlag, composeYAML, 0o600); err != nil {
		logger.Error("failed to write docker-compose.yml: %s", err)
		os.Exit(1)
	}
}
