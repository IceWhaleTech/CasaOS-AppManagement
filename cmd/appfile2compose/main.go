package main

import (
	"encoding/json"
	"os"

	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"gopkg.in/yaml.v3"
)

var logger = NewLogger()

func main() {
	if len(os.Args) < 2 {
		os.Args = append(os.Args, "-h")
	}

	inputFlag := os.Args[1]

	if inputFlag == "" || inputFlag == "-h" || inputFlag == "--help" {
		println("Usage: appfile2compose <appfile.json> [docker-compose.yml]")
		os.Exit(0)
	}

	file, err := os.Open(inputFlag)
	if err != nil {
		logger.Error("%s", err.Error())
		os.Exit(1)
	}

	decoder := json.NewDecoder(file)

	var appFile model.CustomizationPostData
	if err := decoder.Decode(&appFile); err != nil {
		logger.Error("failed to decode appfile %s: %s", inputFlag, err.Error())
		os.Exit(1)
	}

	composeApp := appFile.Compose()

	composeYAML, err := yaml.Marshal(composeApp)
	if err != nil {
		logger.Error("failed to marshal compose app converted from appfile %s: %s", inputFlag, err.Error())
		os.Exit(1)
	}

	if len(os.Args) < 3 {
		println(string(composeYAML))
		os.Exit(0)
	}

	outputFlag := os.Args[2]

	if err := os.WriteFile(outputFlag, composeYAML, 0o600); err != nil {
		logger.Error("failed to write %s: %s", outputFlag, err.Error())
		os.Exit(1)
	}
}
