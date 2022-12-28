package v2

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-Common/utils/random"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	"go.uber.org/zap"
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

func (a *ComposeApp) PrepareInstall() (*ComposeApp, error) {
	if err := fixProjectName((*codegen.ComposeApp)(a)); err != nil {
		logger.Error("failed to fix project name", zap.Error(err))
		return nil, err
	}

	a.Name = a.Name + "-" + random.RandomString(4, true)

	a.WorkingDir = filepath.Join(config.AppInfo.AppsPath, a.Name)

	if err := file.IsNotExistMkDir(a.WorkingDir); err != nil {
		logger.Error("failed to create working dir", zap.Error(err), zap.String("path", a.WorkingDir))
		return nil, err
	}

	// save to workdir
	path := filepath.Join(a.WorkingDir, common.ComposeYAMLFileName)
	if err := os.WriteFile(path, []byte(*a.YAML()), 0o600); err != nil {
		logger.Error("failed to save compose file", zap.Error(err), zap.String("path", path))

		if err := file.RMDir(a.WorkingDir); err != nil {
			logger.Error("failed to cleanup working dir after failing to save compose file", zap.Error(err), zap.String("path", a.WorkingDir))
		}
		return nil, err
	}

	return a, nil
}

func (a *ComposeApp) Install() (*ComposeApp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second) // 5 minutes

	go func(ctx context.Context, cancel context.CancelFunc) {
		defer cancel()

		// pull
		if err := a.Pull(ctx, cancel); err != nil {
			logger.Error("failed to pull images for compose app", zap.Error(err))
			return
		}

		// prepare install
		if _, err := a.PrepareInstall(); err != nil {
			logger.Error("failed to prepare install compose app", zap.Error(err))
			return
		}

		// TODO - create

		// TODO - start
	}(ctx, cancel)

	return a, nil
}

func (a *ComposeApp) Pull(ctx context.Context, cancel context.CancelFunc) error {
	service, err := apiService()
	if err != nil {
		return err
	}

	return service.Pull(ctx, (*codegen.ComposeApp)(a), api.PullOptions{})
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

	project.Extensions["yaml"] = &SampleComposeAppYAML

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

func apiService() (api.Service, error) {
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		return nil, err
	}

	if err := dockerCli.Initialize(&flags.ClientOptions{
		Common: &flags.CommonOptions{},
	}); err != nil {
		return nil, err
	}

	return compose.NewComposeService(dockerCli), nil
}
