package v2

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestGetComposeApp(t *testing.T) {
	for storeAppID, composeApp := range Store {
		storeInfo, err := composeApp.StoreInfo()
		assert.NilError(t, err)
		assert.Equal(t, storeInfo.AppStoreID, storeAppID)
	}
}

func TestComposeYAML(t *testing.T) {
	for _, composeApp := range Store {
		assert.Equal(t, *composeApp.YAML(), SampleComposeAppYAML)
	}
}

func TestGetApp(t *testing.T) {
	for _, composeApp := range Store {
		for _, service := range composeApp.Services {
			app := composeApp.App(service.Name)
			assert.Equal(t, app.Name, service.Name)
		}
	}
}

func TestGetMainApp(t *testing.T) {
	for _, composeApp := range Store {
		mainApp, err := composeApp.MainApp()
		assert.NilError(t, err)
		assert.Equal(t, mainApp.Name, "syncthing")
	}
}
