package v2

import (
	"bytes"
	"context"
	"path/filepath"
	"strconv"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	composeCmd "github.com/docker/compose/v2/cmd/compose"
	"github.com/docker/compose/v2/cmd/formatter"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
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

func (a *ComposeApp) PullAndInstall(ctx context.Context) error {
	service, err := apiService()
	if err != nil {
		return err
	}

	if err := service.Pull(ctx, (*codegen.ComposeApp)(a), api.PullOptions{}); err != nil {
		return err
	}

	// prepare source path for volumes if not exist
	for _, app := range a.Services {
		for _, volume := range app.Volumes {
			path := volume.Source
			if err := file.IsNotExistMkDir(path); err != nil {
				return err
			}
		}
	}

	if err := service.Create(ctx, (*codegen.ComposeApp)(a), api.CreateOptions{}); err != nil {
		return err
	}

	if err := service.Start(ctx, a.Name, api.StartOptions{
		CascadeStop: true,
		Wait:        true,
	}); err != nil {
		return err
	}

	return service.Up(ctx, (*codegen.ComposeApp)(a), api.UpOptions{})
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

func LoadComposeAppFromConfigFile(appID string, configFile string) (*ComposeApp, error) {
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
	)

	return (*ComposeApp)(project), err
}

func NewComposeAppFromYAML(yaml []byte, env map[string]string) (*ComposeApp, error) {
	composeApp, err := loader.Load(
		types.ConfigDetails{
			ConfigFiles: []types.ConfigFile{
				{
					Content: []byte(yaml),
				},
			},
			Environment: lo.If(env == nil, map[string]string{}).Else(env),
		},
		func(o *loader.Options) { o.SkipInterpolation = (len(env) == 0) },
	)
	if err != nil {
		return nil, err
	}

	// populate yaml in extensions
	if composeApp.Extensions == nil {
		composeApp.Extensions = make(map[string]interface{})
	}

	// fix name
	if err := fixName(composeApp); err != nil {
		return nil, err
	}

	return (*ComposeApp)(composeApp), nil
}

func getNameFromYAML(composeYAML []byte) (string, error) {
	var baseStructure struct {
		Name string `yaml:"name"`
	}

	if err := yaml.Unmarshal(composeYAML, &baseStructure); err != nil {
		return "", err
	}

	return baseStructure.Name, nil
}

func fixName(composeApp *codegen.ComposeApp) error {
	if composeApp.Name == "" {
		_composeApp := (*ComposeApp)(composeApp)
		storeInfo, err := _composeApp.StoreInfo(false)
		if err != nil {
			return err
		}
		composeApp.Name = *storeInfo.MainApp
	}

	return nil
}
