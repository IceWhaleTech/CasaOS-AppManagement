package v2_test

import (
	"testing"

	v2 "github.com/IceWhaleTech/CasaOS-AppManagement/service/v2"
	"gotest.tools/v3/assert"
)

func TestGetComposeApp(t *testing.T) {
	appStore, err := v2.NewAppStore()
	assert.NilError(t, err)

	for storeAppID, composeApp := range appStore.Catalog() {
		storeInfo, err := composeApp.StoreInfo()
		assert.NilError(t, err)
		assert.Equal(t, *storeInfo.AppStoreID, storeAppID)
	}
}

func TestComposeYAML(t *testing.T) {
	appStore, err := v2.NewAppStore()
	assert.NilError(t, err)

	for _, composeApp := range appStore.Catalog() {
		assert.Equal(t, *composeApp.YAML(), v2.SampleComposeAppYAML)
	}
}

func TestGetApp(t *testing.T) {
	appStore, err := v2.NewAppStore()
	assert.NilError(t, err)

	for _, composeApp := range appStore.Catalog() {
		for _, service := range composeApp.Services {
			app := composeApp.App(service.Name)
			assert.Equal(t, app.Name, service.Name)
		}
	}
}
