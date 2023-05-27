package model_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

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
