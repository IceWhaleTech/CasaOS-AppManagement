package v2

import (
	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
)

type ComposeApp types.Project

func (a *ComposeApp) StoreInfo() (*codegen.ComposeAppStoreInfo, error) {
	if ex, ok := a.Extensions[common.ComposeYamlExtensionName]; ok {
		var storeInfo codegen.ComposeAppStoreInfo
		if err := loader.Transform(ex, &storeInfo); err != nil {
			return nil, err
		}
		return &storeInfo, nil
	}
	return nil, ErrYAMLExtensionNotFound
}

func (a *ComposeApp) YAML() *string {
	if yaml, ok := a.Extensions["yaml"]; ok {
		return yaml.(*string)
	}
	return nil
}

func (a *ComposeApp) App(name string) *App {
	for i, service := range a.Services {
		if service.Name == name {
			return (*App)(&a.Services[i])
		}
	}

	return nil
}

func (a *ComposeApp) Apps() map[string]*App {
	apps := make(map[string]*App)

	for i, service := range a.Services {
		apps[service.Name] = (*App)(&a.Services[i])
	}

	return apps
}

func (a *ComposeApp) MainApp() (*App, error) {
	storeInfo, err := a.StoreInfo()
	if err != nil {
		return nil, err
	}

	if storeInfo.MainApp == nil || *storeInfo.MainApp == "" {
		return (*App)(&a.Services[0]), nil
	}

	app := a.App(*storeInfo.MainApp)
	if app == nil {
		return nil, ErrMainAppNotFound
	}

	return app, nil
}
