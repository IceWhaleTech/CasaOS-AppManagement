package service

import (
	"path/filepath"
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"go.uber.org/goleak"
	"gotest.tools/v3/assert"
)

func TestGetComposeApp(t *testing.T) {
	defer goleak.VerifyNone(t)

	appStore, err := NewAppStore("https://github.com/IceWhaleTech/CasaOS-AppStore/archive/refs/heads/main.zip")
	assert.NilError(t, err)

	for storeAppID, composeApp := range appStore.Catalog() {
		storeInfo, err := composeApp.StoreInfo(true)
		assert.NilError(t, err)
		assert.Equal(t, *storeInfo.StoreAppID, storeAppID)
	}
}

func TestGetApp(t *testing.T) {
	defer goleak.VerifyNone(t)

	appStore, err := NewAppStore("https://github.com/IceWhaleTech/CasaOS-AppStore/archive/refs/heads/main.zip")
	assert.NilError(t, err)

	for _, composeApp := range appStore.Catalog() {
		for _, service := range composeApp.Services {
			app := composeApp.App(service.Name)
			assert.Equal(t, app.Name, service.Name)
		}
	}
}

func TestWorkDir(t *testing.T) {
	defer goleak.VerifyNone(t)

	// test for http
	hostport := "localhost:8080"
	appStore, err := NewAppStore("http://" + hostport)
	assert.NilError(t, err)

	workdir, err := appStore.WorkDir()
	assert.NilError(t, err)
	assert.Equal(t, workdir, filepath.Join(config.AppInfo.AppStorePath, hostport, "d41d8cd98f00b204e9800998ecf8427e"))

	// test for https
	appStore, err = NewAppStore("https://" + hostport)
	assert.NilError(t, err)

	workdir, err = appStore.WorkDir()
	assert.NilError(t, err)
	assert.Equal(t, workdir, filepath.Join(config.AppInfo.AppStorePath, hostport, "d41d8cd98f00b204e9800998ecf8427e"))

	// test for github
	hostname := "github.com"
	path := "/IceWhaleTech/CasaOS-AppStore/archive/refs/heads/main.zip"
	appStore, err = NewAppStore("https://" + hostname + path)
	assert.NilError(t, err)

	workdir, err = appStore.WorkDir()
	assert.NilError(t, err)
	assert.Equal(t, workdir, filepath.Join(config.AppInfo.AppStorePath, hostname, "8b0968a7d7cda3f813d05736a89d0c92"))
}
