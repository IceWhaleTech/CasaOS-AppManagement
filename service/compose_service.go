package service

import (
	"context"
	"os"
	"path/filepath"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	timeutils "github.com/IceWhaleTech/CasaOS-Common/utils/time"
	"gopkg.in/yaml.v3"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"

	"go.uber.org/zap"
)

type ComposeService struct{}

func (s *ComposeService) PrepareWorkingDirectory(name string, composeYAML []byte) (string, error) {
	workingDirectory := filepath.Join(config.AppInfo.AppsPath, name)

	if err := file.IsNotExistMkDir(workingDirectory); err != nil {
		logger.Error("failed to create working dir", zap.Error(err), zap.String("path", workingDirectory))
		return "", err
	}

	return workingDirectory, nil
}

func (s *ComposeService) Install(ctx context.Context, composeYAML []byte) error {
	appID, err := getNameFromYAML(composeYAML)
	if err != nil {
		return err
	}

	// update interpolation map in current context
	interpolationMap := baseInterpolationMap()
	interpolationMap["AppID"] = appID

	// load compose app with env variable interpolation
	composeApp, err := NewComposeAppFromYAML(composeYAML, interpolationMap)
	if err != nil {
		return err
	}

	composeYAMLInterpolated, err := yaml.Marshal(composeApp)
	if err != nil {
		return err
	}

	workingDirectory, err := s.PrepareWorkingDirectory(composeApp.Name, composeYAML)
	if err != nil {
		return err
	}

	yamlFilePath := filepath.Join(workingDirectory, common.ComposeYAMLFileName)

	if err := os.WriteFile(yamlFilePath, composeYAMLInterpolated, 0o600); err != nil {
		logger.Error("failed to save compose file", zap.Error(err), zap.String("path", yamlFilePath))

		if err := file.RMDir(workingDirectory); err != nil {
			logger.Error("failed to cleanup working dir after failing to save compose file", zap.Error(err), zap.String("path", workingDirectory))
		}
		return err
	}

	// load project
	composeApp, err = LoadComposeAppFromConfigFile(composeApp.Name, yamlFilePath)

	if err != nil {
		logger.Error("failed to install compose app", zap.Error(err), zap.String("name", composeApp.Name))
		cleanup(workingDirectory)
		return err
	}

	// prepare for message bus events
	storeInfo, err := composeApp.StoreInfo(true)
	if err != nil {
		return err
	}

	if storeInfo.Apps == nil || len(*storeInfo.Apps) == 0 {
		return ErrNoAppFoundInComposeApp
	}

	mainAppStoreInfo, ok := (*storeInfo.Apps)[*storeInfo.MainApp]
	if !ok {
		return ErrMainAppNotFound
	}

	eventProperties := common.PropertiesFromContext(ctx)
	eventProperties[common.PropertyTypeAppName.Name] = appID
	eventProperties[common.PropertyTypeAppIcon.Name] = mainAppStoreInfo.Icon
	eventProperties[common.PropertyTypeImageName.Name] = composeApp.App(*storeInfo.MainApp).Image

	go func(ctx context.Context) {
		go PublishEventWrapper(ctx, common.EventTypeAppInstallBegin, nil)

		defer PublishEventWrapper(ctx, common.EventTypeAppInstallEnd, nil)

		if err := composeApp.PullAndInstall(ctx); err != nil {
			go PublishEventWrapper(ctx, common.EventTypeAppInstallError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})

			logger.Error("failed to install compose app", zap.Error(err), zap.String("name", composeApp.Name))
		}
	}(ctx)

	return nil
}

func (s *ComposeService) Uninstall(ctx context.Context, composeApp *ComposeApp, deleteConfigFolder bool) error {
	// prepare for message bus events
	storeInfo, err := composeApp.StoreInfo(true)
	if err != nil {
		return err
	}

	if storeInfo.Apps == nil || len(*storeInfo.Apps) == 0 {
		return ErrNoAppFoundInComposeApp
	}

	mainAppStoreInfo, ok := (*storeInfo.Apps)[*storeInfo.MainApp]
	if !ok {
		return ErrMainAppNotFound
	}

	eventProperties := common.PropertiesFromContext(ctx)
	eventProperties[common.PropertyTypeAppName.Name] = composeApp.Name
	eventProperties[common.PropertyTypeAppIcon.Name] = mainAppStoreInfo.Icon

	go func(ctx context.Context) {
		go PublishEventWrapper(ctx, common.EventTypeAppUninstallBegin, nil)

		defer PublishEventWrapper(ctx, common.EventTypeAppUninstallEnd, nil)

		if err := composeApp.Uninstall(ctx, deleteConfigFolder); err != nil {
			go PublishEventWrapper(ctx, common.EventTypeAppUninstallError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})

			logger.Error("failed to uninstall compose app", zap.Error(err), zap.String("name", composeApp.Name))
		}
	}(ctx)

	return nil
}

func (s *ComposeService) Status(ctx context.Context, appID string) (string, error) {
	service, err := apiService()
	if err != nil {
		return "", err
	}

	stackList, err := service.List(ctx, api.ListOptions{
		All: true,
	})
	if err != nil {
		return "", err
	}

	for _, stack := range stackList {
		if stack.ID == appID {
			return stack.Status, nil
		}
	}

	return "", ErrComposeAppNotFound
}

func (s *ComposeService) List(ctx context.Context) (map[string]*ComposeApp, error) {
	service, err := apiService()
	if err != nil {
		return nil, err
	}

	stackList, err := service.List(ctx, api.ListOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	result := map[string]*ComposeApp{}

	for _, stack := range stackList {

		composeApp, err := LoadComposeAppFromConfigFile(stack.ID, stack.ConfigFiles)
		// load project
		if err != nil {
			logger.Error("failed to load compose file", zap.Error(err), zap.String("path", stack.ConfigFiles))
			continue
		}

		result[stack.ID] = composeApp
	}

	return result, nil
}

func NewComposeService() *ComposeService {
	return &ComposeService{}
}

func baseInterpolationMap() map[string]string {
	return map[string]string{
		"DefaultUserName": common.DefaultUserName,
		"DefaultPassword": common.DefaultPassword,
		"PUID":            common.DefaultPUID,
		"PGID":            common.DefaultPGID,
		"TZ":              timeutils.GetSystemTimeZoneName(),
	}
}

func apiService() (api.Service, error) {
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		return nil, err
	}

	if err := dockerCli.Initialize(&flags.ClientOptions{}); err != nil {
		return nil, err
	}

	return compose.NewComposeService(dockerCli), nil
}

func cleanup(workDir string) {
	logger.Info("cleaning up working dir", zap.String("path", workDir))
	if err := file.RMDir(workDir); err != nil {
		logger.Error("failed to cleanup working dir", zap.Error(err), zap.String("path", workDir))
	}
}
