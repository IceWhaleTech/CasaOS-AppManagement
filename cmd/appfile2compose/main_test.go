package main_test

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"

	_logger "github.com/IceWhaleTech/CasaOS-Common/utils/logger"

	main "github.com/IceWhaleTech/CasaOS-AppManagement/cmd/appfile2compose"

	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gopkg.in/yaml.v3"
	"gotest.tools/v3/assert"
)

func TestMain(t *testing.T) {
	_logger.LogInitConsoleOnly()

	appFile, err := main.NewAppFile(filepath.Join("fixtures", "appfile.json"))
	assert.NilError(t, err)

	composeApp1 := appFile.ComposeApp()

	config, err := yaml.Marshal(composeApp1)
	assert.NilError(t, err)

	composeApp2, err := service.NewComposeAppFromYAML(config)
	assert.NilError(t, err)
	assert.Assert(t, composeApp2 != nil)

	err = Compare(composeApp1, composeApp2)
	assert.NilError(t, err)

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

func TestSingle(t *testing.T) {
	t.Skip("Tiger's own test - skip")
	path := "/home/wxh/dev/CasaOS-AppStore/Apps/Vaultwarden/appfile.json"
	validate(t, path)
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

		validate(t, path)
		return nil
	})

	assert.NilError(t, err)
}

func validate(t *testing.T, path string) {
	appFile, err := main.NewAppFile(path)
	assert.NilError(t, err)

	composeApp1 := appFile.ComposeApp()

	config, err := yaml.Marshal(composeApp1)
	assert.NilError(t, err)

	composeApp2, err := service.NewComposeAppFromYAML(config)
	assert.NilError(t, err)
	assert.Assert(t, composeApp2 != nil)

	err = Compare(composeApp1, composeApp2)
	assert.NilError(t, err)

	storeInfo1, err := composeApp1.StoreInfo(true)
	assert.NilError(t, err)

	storeInfo2, err := composeApp2.StoreInfo(true)
	assert.NilError(t, err)

	assert.DeepEqual(t, storeInfo1, storeInfo2, cmpopts.EquateEmpty())

	mainApp1 := composeApp1.App(*storeInfo1.MainApp)
	assert.Assert(t, mainApp1 != nil)

	mainApp2 := composeApp2.App(*storeInfo2.MainApp)
	assert.Assert(t, mainApp2 != nil)

	mainAppStoreInfo1, err := mainApp1.StoreInfo()
	assert.NilError(t, err)

	mainAppStoreInfo2, err := mainApp2.StoreInfo()
	assert.NilError(t, err)

	assert.DeepEqual(t, mainAppStoreInfo1, mainAppStoreInfo2, cmpopts.EquateEmpty())
}

func Compare(composeApp1, composeApp2 *service.ComposeApp) error {
	storeInfo1, err := composeApp1.StoreInfo(true)
	if err != nil {
		return err
	}

	storeInfo2, err := composeApp2.StoreInfo(true)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(storeInfo1, storeInfo2) {
		return fmt.Errorf("store info of two compose apps does not deep equal")
	}

	mainApp1 := composeApp1.App(*storeInfo1.MainApp)
	mainApp2 := composeApp2.App(*storeInfo2.MainApp)

	mainAppStoreInfo1, err := mainApp1.StoreInfo()
	if err != nil {
		return err
	}

	mainAppStoreInfo2, err := mainApp2.StoreInfo()
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(mainAppStoreInfo1, mainAppStoreInfo2) {
		return fmt.Errorf("store info of two main apps does not deep equal")
	}

	return nil
}
