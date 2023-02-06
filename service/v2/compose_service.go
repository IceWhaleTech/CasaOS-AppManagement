package v2

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	timeutils "github.com/IceWhaleTech/CasaOS-Common/utils/time"
	"github.com/samber/lo"

	"github.com/compose-spec/compose-go/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	composeCmd "github.com/docker/compose/v2/cmd/compose"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"

	"go.uber.org/zap"
)

type ComposeService struct {
	ctx context.Context
}

func (s *ComposeService) PrepareWorkingDirectory(name string, composeYAML []byte) (string, error) {
	workingDirectory := filepath.Join(config.AppInfo.AppsPath, name)

	if err := file.IsNotExistMkDir(workingDirectory); err != nil {
		logger.Error("failed to create working dir", zap.Error(err), zap.String("path", workingDirectory))
		return "", err
	}

	yamlFilePath := filepath.Join(workingDirectory, common.ComposeYAMLFileName)
	if err := os.WriteFile(yamlFilePath, composeYAML, 0o600); err != nil {
		logger.Error("failed to save compose file", zap.Error(err), zap.String("path", yamlFilePath))

		if err := file.RMDir(workingDirectory); err != nil {
			logger.Error("failed to cleanup working dir after failing to save compose file", zap.Error(err), zap.String("path", workingDirectory))
		}
		return "", err
	}

	return yamlFilePath, nil
}

func (s *ComposeService) Pull(ctx context.Context, composeApp *ComposeApp) error {
	service, err := apiService()
	if err != nil {
		return err
	}

	return service.Pull(ctx, utils.Ptr(codegen.ComposeApp(*composeApp)), api.PullOptions{})
}

func (s *ComposeService) Install(composeYAML []byte) error {
	composeApp, err := NewComposeAppFromYAML(composeYAML)
	if err != nil {
		return err
	}

	yamlFilePath, err := s.PrepareWorkingDirectory(composeApp.Name, composeYAML)
	if err != nil {
		return err
	}

	options := composeCmd.ProjectOptions{
		ProjectDir:  filepath.Dir(yamlFilePath),
		ProjectName: composeApp.Name,
	}

	// update interpolation map in current context
	interpolationMap := common.InterpolationMapFromContext(s.ctx)
	interpolationMap["AppID"] = composeApp.Name
	ctx := common.WithInterpolationMap(s.ctx, interpolationMap)

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
		cli.WithEnv(lo.MapToSlice(interpolationMap, func(k, v string) string { return k + "=" + v })),
	)
	if err != nil {
		logger.Error("failed to install compose app", zap.Error(err), zap.String("name", composeApp.Name))
		cleanup(yamlFilePath)
		return err
	}

	go func(ctx context.Context) {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()

		if err := pullAndInstall(ctx, project); err != nil {
			logger.Error("failed to install compose app", zap.Error(err), zap.String("name", composeApp.Name))
			cleanup(yamlFilePath)
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

		options := composeCmd.ProjectOptions{
			ProjectDir:  filepath.Dir(stack.ConfigFiles),
			ProjectName: stack.ID,
		}

		// update interpolation map in current context
		interpolationMap := common.InterpolationMapFromContext(s.ctx)
		interpolationMap["AppID"] = stack.ID

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
			cli.WithEnv(lo.MapToSlice(interpolationMap, func(k, v string) string { return k + "=" + v })),
		)
		if err != nil {
			logger.Error("failed to load compose file", zap.Error(err), zap.String("path", stack.ConfigFiles))
			continue
		}

		result[stack.ID] = (*ComposeApp)(project)
	}

	return result, nil
}

func NewComposeService() *ComposeService {
	return &ComposeService{
		ctx: common.WithInterpolationMap(
			context.Background(),
			map[string]string{
				"DefaultUserName": common.DefaultUserName,
				"DefaultPassword": common.DefaultPassword,
				"PUID":            common.DefaultPUID,
				"PGID":            common.DefaultPGID,
				"TZ":              timeutils.GetSystemTimeZoneName(),
			}),
	}
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

func cleanup(workDir string) {
	logger.Info("cleaning up working dir", zap.String("path", workDir))
	if err := file.RMDir(workDir); err != nil {
		logger.Error("failed to cleanup working dir", zap.Error(err), zap.String("path", workDir))
	}
}

func pullAndInstall(ctx context.Context, composeApp *codegen.ComposeApp) error {
	service, err := apiService()
	if err != nil {
		return err
	}

	if err := service.Pull(ctx, composeApp, api.PullOptions{}); err != nil {
		return err
	}

	// prepare source path for volumes if not exist
	for _, app := range composeApp.Services {
		for _, volume := range app.Volumes {
			path := volume.Source
			if err := file.IsNotExistMkDir(path); err != nil {
				return err
			}
		}
	}

	if err := service.Create(ctx, composeApp, api.CreateOptions{}); err != nil {
		return err
	}

	if err := service.Start(ctx, composeApp.Name, api.StartOptions{
		CascadeStop: true,
		Wait:        true,
	}); err != nil {
		return err
	}

	return nil
}
