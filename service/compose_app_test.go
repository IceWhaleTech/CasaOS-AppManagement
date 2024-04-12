package service_test

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"go.uber.org/goleak"
	"gotest.tools/v3/assert"
)

func TestUpdateEventPropertiesFromStoreInfo(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction(topFunc1), goleak.IgnoreTopFunction(pollFunc1), goleak.IgnoreTopFunction(httpFunc1)) // https://github.com/census-instrumentation/opencensus-go/issues/1191

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	logger.LogInitConsoleOnly()

	// mock store compose app
	storeComposeApp, err := service.NewComposeAppFromYAML([]byte(common.SampleComposeAppYAML), true, false)
	assert.NilError(t, err)

	storeInfo, err := storeComposeApp.StoreInfo(false)
	assert.NilError(t, err)

	eventProperties := map[string]string{}
	err = storeComposeApp.UpdateEventPropertiesFromStoreInfo(eventProperties)
	assert.NilError(t, err)

	// icon
	appIcon, ok := eventProperties[common.PropertyTypeAppIcon.Name]
	assert.Assert(t, ok)
	assert.Equal(t, appIcon, storeInfo.Icon)

	// title
	appTitle, ok := eventProperties[common.PropertyTypeAppTitle.Name]
	assert.Assert(t, ok)

	titles := map[string]string{}
	err = json.Unmarshal([]byte(appTitle), &titles)
	assert.NilError(t, err)

	title, ok := titles[common.DefaultLanguage]
	assert.Assert(t, ok)

	assert.Equal(t, title, storeInfo.Title[common.DefaultLanguage])
}

func TestNameAndTitle(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction(topFunc1), goleak.IgnoreTopFunction(pollFunc1), goleak.IgnoreTopFunction(httpFunc1)) // https://github.com/census-instrumentation/opencensus-go/issues/1191

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

func TestUncontrolledApp(t *testing.T) {
	logger.LogInitConsoleOnly()

	app, err := service.NewComposeAppFromYAML([]byte(common.SampleComposeAppYAML), true, false)
	assert.NilError(t, err)

	storeInfo, err := app.StoreInfo(false)
	assert.NilError(t, err)
	// assert nil
	assert.Assert(t, storeInfo.IsUncontrolled == nil)

	err = app.SetUncontrolled(true)
	assert.NilError(t, err)

	storeInfo, err = app.StoreInfo(false)
	assert.NilError(t, err)
	assert.Assert(t, *storeInfo.IsUncontrolled)

	err = app.SetUncontrolled(false)
	assert.NilError(t, err)

	storeInfo, err = app.StoreInfo(false)
	assert.NilError(t, err)
	assert.Assert(t, !*storeInfo.IsUncontrolled)
}
