package v2

import (
	"context"
	"fmt"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

type ComposeApp codegen.ComposeApp

func (a *ComposeApp) StoreInfo() (*codegen.ComposeAppStoreInfo, error) {
	if ex, ok := a.Extensions[common.ComposeExtensionNameXCasaOS]; ok {
		var storeInfo codegen.ComposeAppStoreInfo
		if err := loader.Transform(ex, &storeInfo); err != nil {
			return nil, err
		}

		// locate main app
		if storeInfo.MainApp == nil || *storeInfo.MainApp == "" {
			for _, app := range a.Apps() {
				storeInfo.MainApp = &app.Name
				break
			}
		}

		mainApp := a.App(*storeInfo.MainApp)
		if mainApp == nil {
			return nil, ErrMainAppNotFound
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

	return nil, ErrComposeExtensionNameXCasaOSNotFound
}

func (a *ComposeApp) YAML() *string {
	if _, ok := a.Extensions["yaml"]; !ok {
		out, err := yaml.Marshal(a)
		if err != nil {
			return nil
		}

		a.Extensions["yaml"] = out
	}

	return a.Extensions["yaml"].(*string)
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

func (a *ComposeApp) Containers(ctx context.Context) (map[string]*api.ContainerSummary, error) {
	service, err := apiService()
	if err != nil {
		return nil, err
	}

	containers, err := service.Ps(ctx, a.Name, api.PsOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	containerMap := lo.SliceToMap(
		containers,
		func(c api.ContainerSummary) (string, *api.ContainerSummary) {
			return c.Service, &c
		},
	)

	return containerMap, nil
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

	// populate yaml in extensions
	if project.Extensions == nil {
		project.Extensions = make(map[string]interface{})
	}

	project.Extensions["yaml"] = utils.Ptr(string(yaml))

	// fix name
	if err := fixProjectName(project); err != nil {
		return nil, err
	}

	return (*ComposeApp)(project), nil
}

func fixProjectName(project *codegen.ComposeApp) error {
	if project.Name == "" {
		composeApp := (*ComposeApp)(project)
		storeInfo, err := composeApp.StoreInfo()
		if err != nil {
			return err
		}
		project.Name = *storeInfo.MainApp
	}

	return nil
}
