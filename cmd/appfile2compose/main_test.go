package main

import (
	"path/filepath"
	"testing"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"gopkg.in/yaml.v2"
	"gotest.tools/v3/assert"
)

func TestMain(t *testing.T) {
	appFile, err := NewAppFile(filepath.Join("fixtures", "appfile.json"))
	assert.NilError(t, err)

	composeApp := appFile.ComposeApp()
	assert.Equal(t, composeApp.Name, "jellyfin")

	composeYAML, err := yaml.Marshal(composeApp)
	assert.NilError(t, err)

	project, err := loader.Load(types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{
				Content: composeYAML,
			},
		},
		Environment: map[string]string{},
	})
	assert.NilError(t, err)

	assert.Equal(t, project.Name, "jellyfin")
}
