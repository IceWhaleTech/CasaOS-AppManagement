package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	composeCmd "github.com/docker/compose/v2/cmd/compose"
	"github.com/docker/compose/v2/cmd/formatter"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/samber/lo"
	"go.uber.org/zap"
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

	// check if upgradable
	// upgradable := false
	// if storeInfo.StoreAppID != nil && *storeInfo.StoreAppID != "" {
	// 	storeComposeApp := MyService.V2AppStore().ComposeApp(*storeInfo.StoreAppID)
	// 	if storeComposeApp != nil {
	// 		storeMainApp := storeComposeApp.App(*storeInfo.MainApp)
	// 	}
	// }

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
	service, dockerClient, err := apiService()
	if err != nil {
		return nil, err
	}
	defer dockerClient.Close()

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
	service, dockerClient, err := apiService()
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	// pull
	for _, app := range a.Services {
		if err := func() error {
			go PublishEventWrapper(ctx, common.EventTypeImagePullBegin, map[string]string{
				common.PropertyTypeImageName.Name: app.Image,
			})

			defer PublishEventWrapper(ctx, common.EventTypeImagePullEnd, map[string]string{
				common.PropertyTypeImageName.Name: app.Image,
			})

			if err := docker.PullImage(ctx, app.Image, func(out io.ReadCloser) {
				pullImageProgress(ctx, out, "INSTALL")
			}); err != nil {
				go PublishEventWrapper(ctx, common.EventTypeImagePullError, map[string]string{
					common.PropertyTypeImageName.Name: app.Image,
					common.PropertyTypeMessage.Name:   err.Error(),
				})
			}

			return nil
		}(); err != nil {
			return err
		}
	}

	// create
	if err := func() error {
		go PublishEventWrapper(ctx, common.EventTypeContainerCreateBegin, nil)

		defer PublishEventWrapper(ctx, common.EventTypeContainerCreateEnd, nil)

		// prepare source path for volumes if not exist
		for _, app := range a.Services {
			for _, volume := range app.Volumes {
				path := volume.Source
				if err := file.IsNotExistMkDir(path); err != nil {
					go PublishEventWrapper(ctx, common.EventTypeContainerCreateError, map[string]string{
						common.PropertyTypeMessage.Name: err.Error(),
					})
					return err
				}
			}
		}

		if err := service.Create(ctx, (*codegen.ComposeApp)(a), api.CreateOptions{}); err != nil {
			go PublishEventWrapper(ctx, common.EventTypeContainerCreateError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})
			return err
		}

		return nil
	}(); err != nil {
		return err
	}

	go PublishEventWrapper(ctx, common.EventTypeContainerStartBegin, nil)

	defer PublishEventWrapper(ctx, common.EventTypeContainerStartEnd, nil)

	if err := service.Start(ctx, a.Name, api.StartOptions{
		CascadeStop: true,
		Wait:        true,
	}); err != nil {
		go PublishEventWrapper(ctx, common.EventTypeContainerStartError, map[string]string{
			common.PropertyTypeMessage.Name: err.Error(),
		})
		return err
	}

	return nil
}

func (a *ComposeApp) Uninstall(ctx context.Context, deleteConfigFolder bool) error {
	service, dockerClient, err := apiService()
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	// stop
	if err := func() error {
		go PublishEventWrapper(ctx, common.EventTypeContainerStopBegin, nil)

		defer PublishEventWrapper(ctx, common.EventTypeContainerStopEnd, nil)

		if err := service.Stop(ctx, a.Name, api.StopOptions{}); err != nil {
			go PublishEventWrapper(ctx, common.EventTypeContainerStopError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})

			return err
		}

		return nil
	}(); err != nil {
		return err
	}

	// remove
	go PublishEventWrapper(ctx, common.EventTypeContainerRemoveBegin, nil)

	defer PublishEventWrapper(ctx, common.EventTypeContainerRemoveEnd, nil)

	if err := service.Down(ctx, a.Name, api.DownOptions{
		RemoveOrphans: true,
		Images:        "all",
		Volumes:       true,
	}); err != nil {
		go PublishEventWrapper(ctx, common.EventTypeImageRemoveError, map[string]string{
			common.PropertyTypeMessage.Name: err.Error(),
		})

		return err
	}

	if err := file.RMDir(a.WorkingDir); err != nil {
		go PublishEventWrapper(ctx, common.EventTypeImageRemoveError, map[string]string{
			common.PropertyTypeMessage.Name: err.Error(),
		})
	}

	if !deleteConfigFolder {
		return nil
	}

	for _, app := range a.Services {
		for _, volume := range app.Volumes {
			if strings.Contains(volume.Source, a.Name) {
				path := filepath.Join(strings.Split(volume.Source, a.Name)[0], a.Name)
				if err := file.RMDir(path); err != nil {
					logger.Error("failed to remove compose app config folder", zap.Error(err), zap.String("path", path))

					go PublishEventWrapper(ctx, common.EventTypeImageRemoveError, map[string]string{
						common.PropertyTypeMessage.Name: err.Error(),
					})
				}
			}
		}
	}

	return nil
}

func (a *ComposeApp) UpdateSettings(ctx context.Context, newComposeYAML []byte) error {
	// update interpolation map in current context
	interpolationMap := baseInterpolationMap()
	interpolationMap["AppID"] = a.Name

	// compare new ComposeApp with current ComposeApp
	if getNameFrom(newComposeYAML) != a.Name {
		return ErrComposeAppNotMatch
	}

	if len(a.ComposeFiles) <= 0 {
		return ErrComposeFileNotFound
	}

	if len(a.ComposeFiles) > 1 {
		logger.Info("warning: multiple compose files found, only the first one will be used", zap.String("compose files", strings.Join(a.ComposeFiles, ",")))
	}

	// backup current compose file
	currentComposeFile := a.ComposeFiles[0]

	backupComposeFile := currentComposeFile + "." + "bak"
	if err := file.CopySingleFile(currentComposeFile, backupComposeFile, ""); err != nil {
		logger.Error("failed to backup compose file", zap.Error(err), zap.String("src", currentComposeFile), zap.String("dst", backupComposeFile))
	}

	// start compose app
	service, dockerClient, err := apiService()
	if err != nil {
		return err
	}
	defer dockerClient.Close()

	success := false
	defer func() {
		if !success {
			if err := file.CopySingleFile(backupComposeFile, currentComposeFile, ""); err != nil {
				logger.Error("failed to restore compose file", zap.Error(err), zap.String("src", backupComposeFile), zap.String("dst", currentComposeFile))
			}

			if err := service.Up(ctx, (*codegen.ComposeApp)(a), api.UpOptions{
				Start: api.StartOptions{
					CascadeStop: true,
					Wait:        true,
				},
			}); err != nil {
				logger.Error("failed to start compose app", zap.Error(err), zap.String("name", a.Name))
			}
		}
	}()

	// save new compose file
	if err := file.WriteToFullPath(newComposeYAML, currentComposeFile, 0o600); err != nil {
		logger.Error("failed to save compose file", zap.Error(err), zap.String("path", currentComposeFile))
		return err
	}

	newComposeApp, err := LoadComposeAppFromConfigFile(a.Name, currentComposeFile)
	if err != nil {
		logger.Error("failed to load compose app from config file", zap.Error(err), zap.String("path", currentComposeFile))
		return err
	}

	if err := service.Up(ctx, (*codegen.ComposeApp)(newComposeApp), api.UpOptions{
		Start: api.StartOptions{
			CascadeStop: true,
			Wait:        true,
		},
	}); err != nil {
		logger.Error("failed to start compose app", zap.Error(err), zap.String("name", a.Name))
		return err
	}

	success = true
	return nil
}

func (a *ComposeApp) Logs(ctx context.Context, lines int) ([]byte, error) {
	service, dockerClient, err := apiService()
	if err != nil {
		return nil, err
	}
	defer dockerClient.Close()

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

	env := []string{fmt.Sprintf("%s=%s", "AppID", appID)}
	for k, v := range baseInterpolationMap() {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// load project
	project, err := options.ToProject(
		nil,
		cli.WithOsEnv,
		cli.WithDotEnv,
		cli.WithEnv(env),
		cli.WithConfigFileEnv,
		cli.WithDefaultConfigPath,
		cli.WithEnvFile(options.EnvFile),
		cli.WithName(options.ProjectName),
		cli.WithWorkingDirectory(options.ProjectDir),
	)

	return (*ComposeApp)(project), err
}

func NewComposeAppFromYAML(yaml []byte) (*ComposeApp, error) {
	// env := baseInterpolationMap()
	// appID := getNameFrom(yaml)
	// if appID == "" {
	// 	env["AppID"] = appID
	// }

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

	composeApp := (*ComposeApp)(project)

	if composeApp.Extensions == nil {
		composeApp.Extensions = map[string]interface{}{}
	}

	// fix compose app name
	if composeApp.Name == "" {
		composeAppStoreInfo, err := composeApp.StoreInfo(false)
		if err != nil {
			return nil, err
		}

		composeApp.Name = *composeAppStoreInfo.MainApp
	}

	return composeApp, nil
}

func getNameFrom(composeYAML []byte) string {
	var baseStructure struct {
		Name string `yaml:"name"`
	}

	if err := yaml.Unmarshal(composeYAML, &baseStructure); err != nil {
		return ""
	}

	return baseStructure.Name
}
