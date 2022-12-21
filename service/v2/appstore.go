package v2

import (
	"fmt"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"

	_ "embed"
)

const extensionName = "x-casaos"

var (
	//go:embed fixtures/sample.docker-compose.yaml
	SampleComposeAppYAML string

	Store = map[string]*ComposeApp{}

	ErrExtensionNotFound = fmt.Errorf("extension `%s` not found", extensionName)
	ErrMainAppNotFound   = fmt.Errorf("main app not found")
)

type (
	App types.ServiceConfig
)

func (a *App) StoreInfo() (*codegen.AppStoreInfo, error) {
	if ex, ok := a.Extensions[extensionName]; ok {
		var storeInfo codegen.AppStoreInfo
		if err := loader.Transform(ex, &storeInfo); err != nil {
			return nil, err
		}
		return &storeInfo, nil
	}
	return nil, ErrExtensionNotFound
}

type (
	ComposeApp types.Project
)

func (a *ComposeApp) StoreInfo() (*codegen.ComposeAppStoreInfo, error) {
	if ex, ok := a.Extensions["x-casaos"]; ok {
		var storeInfo codegen.ComposeAppStoreInfo
		if err := loader.Transform(ex, &storeInfo); err != nil {
			return nil, err
		}
		return &storeInfo, nil
	}
	return nil, ErrExtensionNotFound
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

func init() {
	project, err := loader.Load(types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{
				Content: []byte(SampleComposeAppYAML),
			},
		},
	})
	if err != nil {
		panic(err)
	}

	project.Extensions["yaml"] = &SampleComposeAppYAML

	if ex, ok := project.Extensions["x-casaos"]; ok {
		var storeInfo codegen.ComposeAppStoreInfo
		if err := loader.Transform(ex, &storeInfo); err != nil {
			panic(err)
		}

		Store[storeInfo.AppStoreID] = (*ComposeApp)(project)

	} else {
		panic("invalid project extension")
	}
}

func GetComposeApp(appStoreID string) *ComposeApp {
	return Store[appStoreID]
}
