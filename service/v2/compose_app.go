package v2

import (
	"fmt"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
)

type ComposeApp codegen.ComposeApp

func (a *ComposeApp) StoreInfo() (*codegen.ComposeAppStoreInfo, error) {
	if ex, ok := a.Extensions[common.ComposeYamlExtensionName]; ok {
		var storeInfo codegen.ComposeAppStoreInfo
		if err := loader.Transform(ex, &storeInfo); err != nil {
			return nil, err
		}

		// locate main app
		mainApp := a.App(*storeInfo.MainApp)
		if mainApp == nil {
			for _, app := range a.Apps() {
				mainApp = app
				break
			}
		}

		if a.Name == "" {
			a.Name = mainApp.Name
		}

		appStoreInfo, err := mainApp.StoreInfo()
		if err != nil {
			return nil, err
		}

		// appStoreID is auto-generated
		appStoreID := fmt.Sprintf("%s.%s", Standardize(appStoreInfo.Developer), Standardize(a.Name))

		storeInfo.AppStoreID = &appStoreID

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
	if name == "" {
		return nil
	}

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

func (a *ComposeApp) Install() (*ComposeApp, error) {
	// TODO - get workdir

	// TODO - update working dir

	// TODO - generate project name

	// TODO - save to workdir

	// TODO - pull

	// TODO - create

	// TODO - start

	return nil, nil
}

func NewComposeAppFromYAML(yaml []byte) (*ComposeApp, error) {
	project, err := loader.Load(
		types.ConfigDetails{
			ConfigFiles: []types.ConfigFile{
				{
					Content: []byte(yaml),
				},
			},
			Environment: map[string]string{},
		},
		func(o *loader.Options) { o.SkipInterpolation = true },
	)
	if err != nil {
		return nil, err
	}

	project.Extensions["yaml"] = &SampleComposeAppYAML

	return (*ComposeApp)(project), nil
}
