package main

import (
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestMain(t *testing.T) {
	appFile, err := NewAppFile(filepath.Join("fixtures", "appfile.json"))
	assert.NilError(t, err)

	composeApp := appFile.ComposeApp()
	assert.Equal(t, composeApp.Name, "jellyfin")

	composeYAML, err := YAML(composeApp)
	assert.NilError(t, err)

	output := string(composeYAML)
	assert.Assert(t, len(output) > 0)
}
