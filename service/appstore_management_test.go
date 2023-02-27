package service

import (
	"os"
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"go.uber.org/goleak"
	"gotest.tools/v3/assert"
)

func TestAppStoreList(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start")) // https://github.com/census-instrumentation/opencensus-go/issues/1191

	file, err := os.CreateTemp("", "app-management.conf")
	assert.NilError(t, err)

	defer os.Remove(file.Name())

	config.InitSetup(file.Name())

	appStoreManagement := NewAppStoreManagement()

	appStoreList := appStoreManagement.AppStoreList()
	assert.Equal(t, len(appStoreList), 1)

	registeredAppStoreList := []string{}
	appStoreManagement.OnAppStoreRegister(func(appStoreURL string) error {
		registeredAppStoreList = append(registeredAppStoreList, appStoreURL)
		return nil
	})

	unregisteredAppStoreList := []string{}
	appStoreManagement.OnAppStoreUnregister(func(appStoreURL string) error {
		unregisteredAppStoreList = append(unregisteredAppStoreList, appStoreURL)
		return nil
	})

	expectAppStoreURL := "https://appstore.example.com"
	appStoreMetadata, err := appStoreManagement.RegisterAppStore(expectAppStoreURL)
	assert.NilError(t, err)
	assert.Equal(t, *appStoreMetadata.ID, 1)
	assert.Equal(t, *appStoreMetadata.URL, expectAppStoreURL)
	assert.Assert(t, len(registeredAppStoreList) == 1)

	appStoreList = appStoreManagement.AppStoreList()
	assert.Equal(t, len(appStoreList), 2)

	actualAppStoreURL := *appStoreList[1].URL
	assert.Equal(t, actualAppStoreURL, expectAppStoreURL)

	err = appStoreManagement.UnregisterAppStore(1)
	assert.NilError(t, err)
	assert.Assert(t, len(unregisteredAppStoreList) == 1)

	assert.DeepEqual(t, registeredAppStoreList, unregisteredAppStoreList)
}
