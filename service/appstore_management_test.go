package service

import (
	"os"
	"strings"
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"go.uber.org/goleak"
	"gotest.tools/v3/assert"
)

func TestAppStoreList(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start")) // https://github.com/census-instrumentation/opencensus-go/issues/1191

	logger.LogInitConsoleOnly()

	file, err := os.CreateTemp("", "app-management.conf")
	assert.NilError(t, err)

	defer os.Remove(file.Name())

	config.InitSetup(file.Name())
	config.AppInfo.AppStorePath, err = os.MkdirTemp("", "test-app-store-*")
	assert.NilError(t, err)

	defer os.RemoveAll(config.AppInfo.AppStorePath)

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

	expectAppStoreURL := strings.ToLower("https://github.com/IceWhaleTech/CasaOS-AppStore/archive/refs/heads/main.zip")
	ch, err := appStoreManagement.RegisterAppStore(expectAppStoreURL)
	assert.NilError(t, err)

	appStoreMetadata := <-ch
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
