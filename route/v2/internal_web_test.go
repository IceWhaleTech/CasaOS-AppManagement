package v2_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	v2 "github.com/IceWhaleTech/CasaOS-AppManagement/route/v2"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"go.uber.org/goleak"
	"gotest.tools/v3/assert"
)

func TestWebAppGridItemAdapter(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start")) // https://github.com/census-instrumentation/opencensus-go/issues/1191

	defer func() {
		docker.Cache = nil
		runtime.GC()
	}()

	storeRoot, err := os.MkdirTemp("", "internal-web-test-*")
	assert.NilError(t, err)

	defer os.RemoveAll(storeRoot)

	appsPath := filepath.Join(storeRoot, common.AppsDirectoryName)
	err = file.MkDir(appsPath)
	assert.NilError(t, err)

	// build test catalog
	err = file.MkDir(filepath.Join(appsPath, "test1"))
	assert.NilError(t, err)

	composeFilePath := filepath.Join(appsPath, "test1", common.ComposeYAMLFileName)

	err = file.WriteToFullPath([]byte(common.SampleComposeAppYAML), composeFilePath, 0o644)
	assert.NilError(t, err)

	composeApp, err := service.LoadComposeAppFromConfigFile("test1", composeFilePath)
	assert.NilError(t, err)

	storeInfo, err := composeApp.StoreInfo(true)
	assert.NilError(t, err)

	composeAppWithStoreInfo := codegen.ComposeAppWithStoreInfo{
		Compose:   (*codegen.ComposeApp)(composeApp),
		StoreInfo: storeInfo,
		Status:    utils.Ptr("running"),
	}

	gridItem, err := v2.WebAppGridItemAdapter(composeAppWithStoreInfo)
	assert.NilError(t, err)

	assert.Equal(t, *gridItem.Icon, (*storeInfo.Apps)[*storeInfo.MainApp].Icon)
	assert.Equal(t, *gridItem.Image, composeApp.Services[0].Image)
	assert.Equal(t, *gridItem.Hostname, *(*storeInfo.Apps)[*storeInfo.MainApp].Container.Hostname)
	assert.Equal(t, *gridItem.Port, (*storeInfo.Apps)[*storeInfo.MainApp].Container.PortMap)
	assert.Equal(t, *gridItem.Index, (*storeInfo.Apps)[*storeInfo.MainApp].Container.Index)
	assert.Equal(t, *gridItem.Status, "running")
	assert.DeepEqual(t, *gridItem.Title, (*storeInfo.Apps)[*storeInfo.MainApp].Title)
	assert.Equal(t, string(*gridItem.Type), "official")
}
