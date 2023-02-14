package v2

import (
	"bytes"
	"context"
	"path/filepath"
	"strconv"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	composeCmd "github.com/docker/compose/v2/cmd/compose"
	"github.com/docker/compose/v2/cmd/formatter"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/samber/lo"
)

type ComposeApp codegen.ComposeApp

func (a *ComposeApp) StoreInfo(includeApps bool) (*codegen.ComposeAppStoreInfo, error) {
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

	if includeApps {
		apps := map[string]codegen.AppStoreInfo{}

		for _, app := range a.Apps() {
			appStoreInfo, err := app.StoreInfo()
			if err != nil {
				return nil, err
			}
			apps[app.Name] = *appStoreInfo
		}

		storeInfo.Apps = &apps
	}

	return &storeInfo, nil
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

func LoadComposeAppFromConfigFile(appID string, configFile string, env map[string]string) (*ComposeApp, error) {
	options := composeCmd.ProjectOptions{
		ProjectDir:  filepath.Dir(configFile),
		ProjectName: appID,
	}

	// load project
	project, err := options.ToProject(
		nil,
		cli.WithWorkingDirectory(options.ProjectDir),
		cli.WithOsEnv,
		cli.WithEnvFile(options.EnvFile),
		cli.WithDotEnv,
		cli.WithConfigFileEnv,
		cli.WithDefaultConfigPath,
		cli.WithName(options.ProjectName),
		cli.WithEnv(lo.MapToSlice(env, func(k, v string) string { return k + "=" + v })),
	)

	return (*ComposeApp)(project), err
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

	// fix name
	if err := fixProjectName(project); err != nil {
		return nil, err
	}

	return (*ComposeApp)(project), nil
}

func fixProjectName(project *codegen.ComposeApp) error {
	if project.Name == "" {
		composeApp := (*ComposeApp)(project)
		storeInfo, err := composeApp.StoreInfo(false)
		if err != nil {
			return err
		}
		project.Name = *storeInfo.MainApp
	}

	return nil
}
