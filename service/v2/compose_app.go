package v2

import (
	"bytes"
	"context"
	"strconv"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/docker/compose/v2/cmd/formatter"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type ComposeApp codegen.ComposeApp

func (a *ComposeApp) StoreInfo() (*codegen.ComposeAppStoreInfo, error) {
	ex, ok := a.Extensions[common.ComposeExtensionNameXCasaOS]
	if !ok {
		return nil, ErrComposeExtensionNameXCasaOSNotFound
	}

	var storeInfo codegen.ComposeAppStoreInfo
	if err := loader.Transform(ex, &storeInfo); err != nil {
		return nil, err
	}

	// locate main app
	if storeInfo.MainApp == nil || *storeInfo.MainApp == "" {
		// if main app is not specified, use the first app
		for _, app := range a.Apps() {
			storeInfo.MainApp = &app.Name
			break
		}
	}

	// apps
	apps := lo.MapValues(a.Apps(), func(app *App, name string) codegen.AppStoreInfo {
		appStoreInfo, err := app.StoreInfo()
		if err != nil {
			logger.Error("failed to get app store info", zap.Error(err), zap.String("app", name))
			return codegen.AppStoreInfo{}
		}

		return *appStoreInfo
	})

	storeInfo.Apps = &apps

	return &storeInfo, nil
}

func (a *ComposeApp) YAML() (*string, error) {
	if _, ok := a.Extensions[common.ComposeExtensionNameYAML]; !ok {
		out, err := yaml.Marshal(a)
		if err != nil {
			return nil, err
		}

		a.Extensions[common.ComposeExtensionNameYAML] = string(out)
	}

	output, ok := a.Extensions[common.ComposeExtensionNameYAML].(string)
	if !ok {
		return nil, ErrComposeExtensionNameYAMLNotFound
	}

	return &output, nil
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

func (a *ComposeApp) Containers(ctx context.Context) (map[string]api.ContainerSummary, error) {
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
		func(c api.ContainerSummary) (string, api.ContainerSummary) {
			return c.Service, c
		},
	)

	return containerMap, nil
}

func (a *ComposeApp) Logs(ctx context.Context, lines int) ([]byte, error) {
	service, err := apiService()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	consumer := formatter.NewLogConsumer(ctx, &buf, &buf, false, true, false)

	if err := service.Logs(ctx, a.Name, consumer, api.LogOptions{
		Project:  (*codegen.ComposeApp)(a),
		Services: lo.Map(a.Services, func(s types.ServiceConfig, i int) string { return s.Name }),
		Follow:   false,
		Tail:     lo.If(lines < 0, "all").Else(strconv.Itoa(lines)),
	}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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

	project.Extensions[common.ComposeExtensionNameYAML] = string(yaml)

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
