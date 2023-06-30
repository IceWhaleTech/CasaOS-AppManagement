package service_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"go.uber.org/goleak"
	"gopkg.in/yaml.v3"
	"gotest.tools/v3/assert"
)

func TestIsUpgradable(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start")) // https://github.com/census-instrumentation/opencensus-go/issues/1191

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	logger.LogInitConsoleOnly()

	// mock store compose
	storeComposeApp, err := service.NewComposeAppFromYAML([]byte(common.SampleComposeAppYAML), true, false)
	assert.NilError(t, err)

	storeComposeApp.SetStoreAppID("test")

	storeMainAppImage, _ := docker.ExtractImageAndTag(storeComposeApp.Services[0].Image)

	storeComposeAppStoreInfo, err := storeComposeApp.StoreInfo(false)
	assert.NilError(t, err)

	// mock local compose app
	appsPath := t.TempDir()

	composeFilePath := filepath.Join(appsPath, common.ComposeYAMLFileName)

	buf, err := yaml.Marshal(storeComposeApp)
	assert.NilError(t, err)

	err = file.WriteToFullPath(buf, composeFilePath, 0o644)
	assert.NilError(t, err)

	localComposeApp, err := service.LoadComposeAppFromConfigFile(*storeComposeAppStoreInfo.StoreAppID, composeFilePath)
	assert.NilError(t, err)

	upgradable := localComposeApp.IsUpdateAvailableWith(storeComposeApp)
	assert.Assert(t, !upgradable)

	storeComposeApp.Services[0].Image = storeMainAppImage + ":test"

	upgradable = localComposeApp.IsUpdateAvailableWith(storeComposeApp)
	assert.Assert(t, upgradable)
}

func TestNameAndTitle(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start")) // https://github.com/census-instrumentation/opencensus-go/issues/1191

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	logger.LogInitConsoleOnly()

	// mock store compose app
	storeComposeApp, err := service.NewComposeAppFromYAML([]byte(common.SampleVanillaComposeAppYAML), true, false)
	assert.NilError(t, err)

	assert.Assert(t, len(storeComposeApp.Name) > 0)

	storeInfo, err := storeComposeApp.StoreInfo(false)
	assert.NilError(t, err)

	assert.Assert(t, len(storeInfo.Title) > 0)
	assert.Equal(t, storeComposeApp.Name, storeInfo.Title[common.DefaultLanguage])
}
