package service_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"go.uber.org/goleak"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v3"
	"gotest.tools/v3/assert"
)

func TestAppStoreList(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction(topFunc1), goleak.IgnoreTopFunction(pollFunc1), goleak.IgnoreTopFunction(httpFunc1)) // https://github.com/census-instrumentation/opencensus-go/issues/1191

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	logger.LogInitConsoleOnly()

	file, err := os.CreateTemp("", "app-management.conf")
	assert.NilError(t, err)

	defer os.Remove(file.Name())

	config.InitSetup(file.Name(), "")
	config.AppInfo.AppStorePath = t.TempDir()

	appStoreManagement := service.NewAppStoreManagement()

	appStoreList := appStoreManagement.AppStoreList()
	assert.Equal(t, len(appStoreList), 0)

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = common.WithProperties(ctx, map[string]string{})

	expectAppStoreURL := strings.ToLower("https://github.com/IceWhaleTech/_appstore/archive/refs/heads/main.zip")

	ch := make(chan *codegen.AppStoreMetadata)

	err = appStoreManagement.RegisterAppStore(ctx, expectAppStoreURL, func(appStoreMetadata *codegen.AppStoreMetadata) {
		ch <- appStoreMetadata
	})
	assert.NilError(t, err)

	appStoreMetadata := <-ch
	assert.Equal(t, *appStoreMetadata.URL, expectAppStoreURL)
	assert.Assert(t, len(registeredAppStoreList) == 1)

	appStoreList = appStoreManagement.AppStoreList()
	assert.Equal(t, len(appStoreList), 1)

	actualAppStoreURL := *appStoreList[0].URL
	assert.Equal(t, actualAppStoreURL, expectAppStoreURL)

	err = appStoreManagement.UnregisterAppStore(0)
	assert.NilError(t, err)
	assert.Assert(t, len(unregisteredAppStoreList) == 1)

	assert.DeepEqual(t, registeredAppStoreList, unregisteredAppStoreList)
}

func TestIsUpgradable(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction(topFunc1), goleak.IgnoreTopFunction(pollFunc1), goleak.IgnoreTopFunction(httpFunc1)) // https://github.com/census-instrumentation/opencensus-go/issues/1191

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	logger.LogInitConsoleOnly()

	appStoreManagement := service.NewAppStoreManagement()

	// mock store compose app
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

	upgradable, err := appStoreManagement.IsUpdateAvailableWith(localComposeApp, storeComposeApp)
	assert.NilError(t, err)
	assert.Assert(t, !upgradable)

	storeComposeApp.Services[0].Image = storeMainAppImage + ":test"

	upgradable, err = appStoreManagement.IsUpdateAvailableWith(localComposeApp, storeComposeApp)
	assert.NilError(t, err)
	assert.Assert(t, upgradable)
}

// you need docker environment to run this test
// you need zimaos environment to run this test
// run `docker pull correctroad/logseq@sha256:7b09ab360c6d253e38fcff54c7f64c45f46b2a6297fb8b640d57a06160de09c4`
// run `docker tag correctroad/logseq@sha256:7b09ab360c6d253e38fcff54c7f64c45f46b2a6297fb8b640d57a06160de09c4 correctroad/logseq:latest`
// run `docker compose up -d` to start old version
// run the test
func TestLatestAppUpdate(t *testing.T) {
	t.Skip("the test is ingreation testing")

	logger.LogInitConsoleOnly()
	config.InitSetup("", "")
	config.InitGlobal("")

	appStoreManagement := service.NewAppStoreManagement()
	composeApp, err := service.NewComposeAppFromYAML([]byte(common.LatestComposeAppYAML), true, false)
	assert.NilError(t, err)

	composeApp.SetStoreAppID("logseq")

	updateAvailable := appStoreManagement.IsUpdateAvailable(composeApp)
	assert.Equal(t, false, updateAvailable)
}

// you need docker environment to run this test
// you need zimaos environment to run this test
// run `docker pull johnguan/stable-diffusion-webui:latest`
// run `docker compose up -d` to start old version
// run the test
func TestSDAppUpdate(t *testing.T) {
	t.Skip("the test is ingreation testing")

	logger.LogInitConsoleOnly()
	config.InitSetup("", "")
	config.InitGlobal("")

	appStoreManagement := service.NewAppStoreManagement()
	composeApp, err := service.NewComposeAppFromYAML([]byte(common.SDComposeAppYAML), true, false)
	assert.NilError(t, err)

	composeApp.SetStoreAppID("stable-diffusion-webui")

	updateAvailable := appStoreManagement.IsUpdateAvailable(composeApp)
	assert.Equal(t, false, updateAvailable)
}

// you need docker environment to run this test
func TestCompareDigest(t *testing.T) {
	t.Skip("the test is ingreation testing")

	match, err := docker.CompareDigest("johnguan/stable-diffusion-webui:latest", []string{"johnguan/stable-diffusion-webui@sha256:9f147d4995464dda8c9e625be91e21ce553d1617e95cb0ebcf23be40e840063b"})
	assert.NilError(t, err)
	assert.Equal(t, true, match)

	match, err = docker.CompareDigest("neosmemo/memos:stable", []string{""})
	assert.NilError(t, err)
	assert.Equal(t, false, match)
}

// you need docker environment to run this test
// you need zimaos environment to run this test
// run `docker pull correctroad/logseq@sha256:7b09ab360c6d253e38fcff54c7f64c45f46b2a6297fb8b640d57a06160de09c4`
// run `docker tag correctroad/logseq@sha256:7b09ab360c6d253e38fcff54c7f64c45f46b2a6297fb8b640d57a06160de09c4 correctroad/logseq:latest`
// run `docker compose up -d` to start old version
// run the test
func TestUpdateLogseqToLatestDocker(t *testing.T) {
	t.Skip("the test is ingreation testing")

	logger.LogInitConsoleOnly()
	config.InitSetup("", "")
	config.InitGlobal("")

	service.MyService = service.NewService("/var/run/casaos")
	composeApp, err := service.NewComposeAppFromYAML([]byte(common.LatestComposeAppYAML), true, false)
	assert.NilError(t, err)

	// TODO: assert logseq digest is sha256:7b09ab360c6d253e38fcff54c7f64c45f46b2a6297fb8b640d57a06160de09c4

	composeApp.SetStoreAppID("logseq")
	dockerPath := filepath.Join(t.TempDir(), "docker-compose.yml")
	err = file.WriteToFullPath([]byte(common.LatestComposeAppYAML), dockerPath, 0o644)
	assert.NilError(t, err)

	composeApp.ComposeFiles = []string{dockerPath}

	err = composeApp.Update(context.Background())
	assert.NilError(t, err)

	select {}
	// docker images --digests
	// TODO: to check the digest is changed
}
