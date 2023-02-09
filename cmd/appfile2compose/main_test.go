package main

import (
	"io/fs"
	"path/filepath"
	"testing"

	v2 "github.com/IceWhaleTech/CasaOS-AppManagement/service/v2"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gopkg.in/yaml.v3"
	"gotest.tools/v3/assert"
)

func TestMain(t *testing.T) {
	appFile, err := NewAppFile(filepath.Join("fixtures", "appfile.json"))
	assert.NilError(t, err)

	composeApp1 := appFile.ComposeApp()

	config, err := yaml.Marshal(composeApp1)
	assert.NilError(t, err)

	composeApp2, err := v2.NewComposeAppFromYAML(config)
	assert.NilError(t, err)
	assert.Assert(t, composeApp2 != nil)

	storeInfo1, err := composeApp1.StoreInfo(true)
	assert.NilError(t, err)

	storeInfo2, err := composeApp2.StoreInfo(true)
	assert.NilError(t, err)

	assert.DeepEqual(t, storeInfo1, storeInfo2)

	mainApp1 := composeApp1.App(*storeInfo1.MainApp)
	assert.Assert(t, mainApp1 != nil)

	mainApp2 := composeApp2.App(*storeInfo2.MainApp)
	assert.Assert(t, mainApp2 != nil)

	mainAppStoreInfo1, err := mainApp1.StoreInfo()
	assert.NilError(t, err)

	mainAppStoreInfo2, err := mainApp2.StoreInfo()
	assert.NilError(t, err)

	assert.DeepEqual(t, mainAppStoreInfo1, mainAppStoreInfo2)
}

func TestAll(t *testing.T) {
	t.Skip("Tiger's own test - skip")
	appsRootDir := "/home/wxh/dev/CasaOS-AppStore/Apps"

	err := filepath.WalkDir(appsRootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Base(path) != "appfile.json" {
			return nil
		}

		appFile, err := NewAppFile(path)
		assert.NilError(t, err)

		composeApp1 := appFile.ComposeApp()

		config, err := yaml.Marshal(composeApp1)
		assert.NilError(t, err)

		composeApp2, err := v2.NewComposeAppFromYAML(config)
		assert.NilError(t, err)
		assert.Assert(t, composeApp2 != nil)

		storeInfo1, err := composeApp1.StoreInfo(true)
		assert.NilError(t, err)

		storeInfo2, err := composeApp2.StoreInfo(true)
		assert.NilError(t, err)

		assert.DeepEqual(t, storeInfo1, storeInfo2)

		mainApp1 := composeApp1.App(*storeInfo1.MainApp)
		assert.Assert(t, mainApp1 != nil)

		mainApp2 := composeApp2.App(*storeInfo2.MainApp)
		assert.Assert(t, mainApp2 != nil)

		mainAppStoreInfo1, err := mainApp1.StoreInfo()
		assert.NilError(t, err)

		mainAppStoreInfo2, err := mainApp2.StoreInfo()
		assert.NilError(t, err)

		assert.DeepEqual(t, mainAppStoreInfo1, mainAppStoreInfo2, cmpopts.EquateEmpty())

		return nil
	})

	assert.NilError(t, err)
}
