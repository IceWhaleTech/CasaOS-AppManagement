package model_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

var ConfigContent = `[common]
RuntimePath = /var/run/casaos

[app]
LogPath = /var/log/casaos/
LogSaveName = app-management
LogFileExt = log
DBPath     = /var/lib/casaos/db
AppStorePath = /var/lib/casaos/appstore
AppsPath = /var/lib/casaos/apps
OpenAIAPIKey = [please_input_your_openai_api_key_in_here_like_sk-xxxx]

[server]
appstore = https://github.com/IceWhaleTech/_appstore/archive/refs/heads/main.zip
`

func TestCompose(t *testing.T) {
	var legacyApp model.CustomizationPostData

	err := json.Unmarshal([]byte(common.SampleLegacyAppfileExportJSON), &legacyApp)
	assert.NoError(t, err)

	compose := legacyApp.Compose()
	assert.Equal(t, strings.ToLower(legacyApp.ContainerName), compose.Name)

	tmpDir, err := os.MkdirTemp("", "test-compose-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	buf, err := yaml.Marshal(compose)
	assert.NoError(t, err)

	yamlFilePath := filepath.Join(tmpDir, "docker-compose.yaml")
	err = os.WriteFile(yamlFilePath, buf, 0o600)
	assert.NoError(t, err)

	composeApp, err := service.LoadComposeAppFromConfigFile("test", yamlFilePath)
	assert.NoError(t, err)

	assert.NotNil(t, composeApp)
}

func TestOpenAIApiKey(t *testing.T) {

	// create and init config
	file, err := os.CreateTemp("", "app-management.conf")
	_, err = file.WriteString(ConfigContent)
	assert.NoError(t, err)
	defer os.Remove(file.Name())

	config.InitSetup(file.Name())

	// create a compose app
	var legacyApp model.CustomizationPostData
	err = json.Unmarshal([]byte(common.SampleLegacyAppfileExportJSON), &legacyApp)
	assert.NoError(t, err)

	compose := legacyApp.Compose()
	assert.Equal(t, strings.ToLower(legacyApp.ContainerName), compose.Name)

	tmpDir, err := os.MkdirTemp("", "test-compose-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	buf, err := yaml.Marshal(compose)
	assert.NoError(t, err)

	yamlFilePath := filepath.Join(tmpDir, "docker-compose.yaml")
	err = os.WriteFile(yamlFilePath, buf, 0o600)
	assert.NoError(t, err)

	composeApp, err := service.LoadComposeAppFromConfigFile("test", yamlFilePath)
	assert.NoError(t, err)

	assert.NotNil(t, composeApp)
	assert.NoError(t, err)

	assert.Equal(t, composeApp.Environment["OPENAI_API_KEY"], "[please_input_your_openai_api_key_in_here_like_sk-xxxx]")
}
