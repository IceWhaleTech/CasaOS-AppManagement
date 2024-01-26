package service_test

import (
	_ "embed"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/samber/lo"
	"go.uber.org/goleak"
	"gotest.tools/v3/assert"
)

func TestGetComposeApp(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction(topFunc1), goleak.IgnoreTopFunction(pollFunc1), goleak.IgnoreTopFunction(httpFunc1)) // https://github.com/census-instrumentation/opencensus-go/issues/1191
	logger.LogInitConsoleOnly()

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	appStorePath, err := os.MkdirTemp("", "appstore")
	assert.NilError(t, err)

	defer os.RemoveAll(appStorePath)

	config.AppInfo.AppStorePath = appStorePath

	appStore, err := service.AppStoreByURL("https://github.com/IceWhaleTech/_appstore/archive/refs/heads/main.zip")
	assert.NilError(t, err)

	err = appStore.UpdateCatalog()
	assert.NilError(t, err)

	catalog, err := appStore.Catalog()
	assert.NilError(t, err)

	for name, composeApp := range catalog {
		assert.Equal(t, name, composeApp.Name)
	}
}

func TestGetApp(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction(topFunc1), goleak.IgnoreTopFunction(pollFunc1), goleak.IgnoreTopFunction(httpFunc1)) // https://github.com/census-instrumentation/opencensus-go/issues/1191

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	logger.LogInitConsoleOnly()

	appStorePath, err := os.MkdirTemp("", "appstore")
	assert.NilError(t, err)

	defer os.RemoveAll(appStorePath)

	config.AppInfo.AppStorePath = appStorePath

	appStore, err := service.AppStoreByURL("https://github.com/IceWhaleTech/_appstore/archive/refs/heads/main.zip")
	assert.NilError(t, err)

	err = appStore.UpdateCatalog()
	assert.NilError(t, err)

	catalog, err := appStore.Catalog()
	assert.NilError(t, err)

	for _, composeApp := range catalog {
		for _, service := range composeApp.Services {
			app := composeApp.App(service.Name)
			assert.Equal(t, app.Name, service.Name)
		}
	}
}

// Note: the test need root permission
func TestSkipUpdateCatalog(t *testing.T) {
	logger.LogInitConsoleOnly()

	appStoreUrl := []string{
		"https://casaos.app/store/main.zip",
		"https://casaos.oss-cn-shanghai.aliyuncs.com/store/main.zip",
	}

	for _, url := range appStoreUrl {
		appStore, err := service.AppStoreByURL(url)
		assert.NilError(t, err)
		workdir, err := appStore.WorkDir()
		assert.NilError(t, err)

		// mkdir workdir for first
		err = file.MkDir(workdir)
		assert.NilError(t, err)

		appStoreStat, err := os.Stat(workdir)
		assert.NilError(t, err)

		err = appStore.UpdateCatalog()
		assert.NilError(t, err)

		// get create and change time of appstore
		appStoreStat_first, err := os.Stat(workdir)
		assert.NilError(t, err)

		assert.Equal(t, false, appStoreStat_first.ModTime().Equal(appStoreStat.ModTime()))

		err = appStore.UpdateCatalog()
		assert.NilError(t, err)

		// get create and change time of appstore
		appStoreStat_second, err := os.Stat(workdir)
		assert.NilError(t, err)

		assert.Equal(t, appStoreStat_first.ModTime(), appStoreStat_second.ModTime())
	}
}

func TestWorkDir(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction(topFunc1), goleak.IgnoreTopFunction(pollFunc1), goleak.IgnoreTopFunction(httpFunc1)) // https://github.com/census-instrumentation/opencensus-go/issues/1191

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	// test for http
	hostport := "localhost:8080"
	appStore, err := service.AppStoreByURL("http://" + hostport)
	assert.NilError(t, err)

	workdir, err := appStore.WorkDir()
	assert.NilError(t, err)
	assert.Equal(t, workdir, filepath.Join(config.AppInfo.AppStorePath, hostport, "d41d8cd98f00b204e9800998ecf8427e"))

	// test for https
	appStore, err = service.AppStoreByURL("https://" + hostport)
	assert.NilError(t, err)

	workdir, err = appStore.WorkDir()
	assert.NilError(t, err)
	assert.Equal(t, workdir, filepath.Join(config.AppInfo.AppStorePath, hostport, "d41d8cd98f00b204e9800998ecf8427e"))

	// test for github
	hostname := "github.com"
	path := "/IceWhaleTech/CasaOS-AppStore/archive/refs/heads/main.zip"
	appStore, err = service.AppStoreByURL("https://" + hostname + path)
	assert.NilError(t, err)

	workdir, err = appStore.WorkDir()
	assert.NilError(t, err)
	assert.Equal(t, workdir, filepath.Join(config.AppInfo.AppStorePath, hostname, "8b0968a7d7cda3f813d05736a89d0c92"))
}

func TestStoreRoot(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction(topFunc1), goleak.IgnoreTopFunction(pollFunc1), goleak.IgnoreTopFunction(httpFunc1)) // https://github.com/census-instrumentation/opencensus-go/issues/1191

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	workdir := t.TempDir()

	expectedStoreRoot := filepath.Join(workdir, "github.com", "IceWhaleTech", "CasaOS-AppStore", "main")
	err := file.MkDir(filepath.Join(expectedStoreRoot, common.AppsDirectoryName))
	assert.NilError(t, err)

	actualStoreRoot, err := service.StoreRoot(workdir)
	assert.NilError(t, err)

	assert.Equal(t, actualStoreRoot, expectedStoreRoot)
}

func TestLoadCategoryList(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction(topFunc1), goleak.IgnoreTopFunction(pollFunc1), goleak.IgnoreTopFunction(httpFunc1)) // https://github.com/census-instrumentation/opencensus-go/issues/1191

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	logger.LogInitConsoleOnly()

	storeRoot := t.TempDir()

	categoryListFilePath := filepath.Join(storeRoot, common.CategoryListFileName)

	err := file.WriteToFullPath([]byte(common.SampleCategoryListJSON), categoryListFilePath, 0o644)
	assert.NilError(t, err)

	dummyList := []interface{}{}
	buf := file.ReadFullFile(categoryListFilePath)
	err = json.Unmarshal(buf, &dummyList)
	assert.NilError(t, err)

	actualCategoryMap := service.LoadCategoryMap(storeRoot)
	assert.Assert(t, actualCategoryMap != nil)
	assert.Equal(t, len(actualCategoryMap), len(dummyList))

	for name, category := range actualCategoryMap {
		assert.Assert(t, category.Name != nil)
		assert.Assert(t, *category.Name == name)

		assert.Assert(t, category.Font != nil)
		assert.Assert(t, *category.Font != "")

		assert.Assert(t, category.Description != nil)
	}
}

func TestLoadRecommend(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction(topFunc1), goleak.IgnoreTopFunction(pollFunc1), goleak.IgnoreTopFunction(httpFunc1)) // https://github.com/census-instrumentation/opencensus-go/issues/1191

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	logger.LogInitConsoleOnly()

	storeRoot := t.TempDir()

	recommendListFilePath := filepath.Join(storeRoot, common.RecommendListFileName)

	type recommendListItem struct {
		AppID string `json:"appid"`
	}

	expectedRecommendList := []recommendListItem{
		{AppID: "app1"},
		{AppID: "app2"},
		{AppID: "app3"},
	}
	buf, err := json.Marshal(expectedRecommendList)
	assert.NilError(t, err)

	err = file.WriteToFullPath(buf, recommendListFilePath, 0o644)
	assert.NilError(t, err)

	actualRecommendList := service.LoadRecommend(storeRoot)
	assert.DeepEqual(t, actualRecommendList, lo.Map(expectedRecommendList, func(item recommendListItem, i int) string {
		return item.AppID
	}))
}

func TestBuildCatalog(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction(topFunc1), goleak.IgnoreTopFunction(pollFunc1), goleak.IgnoreTopFunction(httpFunc1)) // https://github.com/census-instrumentation/opencensus-go/issues/1191

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	storeRoot := t.TempDir()

	// test for invalid storeRoot
	_, err := service.BuildCatalog(storeRoot)
	assert.ErrorType(t, err, new(fs.PathError))

	appsPath := filepath.Join(storeRoot, common.AppsDirectoryName)
	err = file.MkDir(appsPath)
	assert.NilError(t, err)

	// test for empty catalog
	catalog, err := service.BuildCatalog(storeRoot)
	assert.NilError(t, err)
	assert.Equal(t, len(catalog), 0)

	// build test catalog
	err = file.MkDir(filepath.Join(appsPath, "test1"))
	assert.NilError(t, err)

	err = file.WriteToFullPath([]byte(common.SampleComposeAppYAML), filepath.Join(appsPath, "test1", common.ComposeYAMLFileName), 0o644)
	assert.NilError(t, err)
	catalog, err = service.BuildCatalog(storeRoot)
	assert.NilError(t, err)
	assert.Equal(t, len(catalog), 1)
}
