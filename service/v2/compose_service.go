package v2

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-Common/utils/random"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	composeCmd "github.com/docker/compose/v2/cmd/compose"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	"go.uber.org/zap"
)

type ComposeService struct{}

func (s *ComposeService) PrepareWorkingDirectory(projectName string, composeYAML []byte) (string, error) {
	name := projectName + "-" + random.RandomString(4, true)
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

func (s *ComposeService) Pull(ctx context.Context, composeApp *codegen.ComposeApp) error {
	service, err := apiService()
	if err != nil {
		return err
	}

	return service.Pull(ctx, composeApp, api.PullOptions{})
}

func (s *ComposeService) Install(projectName string, composeYAML []byte) (*codegen.ComposeApp, error) {
	yamlFilePath, err := s.PrepareWorkingDirectory(projectName, composeYAML)
	if err != nil {
		return nil, err
	}

	options := composeCmd.ProjectOptions{
		ConfigPaths: []string{yamlFilePath},
		WorkDir:     filepath.Dir(yamlFilePath),
		ProjectDir:  filepath.Dir(yamlFilePath),
		ProjectName: projectName,
	}

	composeApp, err := options.ToProject(nil)
	if err != nil {
		logger.Error("failed to create project", zap.Error(err))
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second) // 5 min
	go func(ctx context.Context, cancel context.CancelFunc, composeApp *codegen.ComposeApp) {
		defer cancel()

		if err := s.Pull(ctx, composeApp); err != nil {
			logger.Error("failed to pull images", zap.Error(err))
			cleanup(options.WorkDir)
			return
		}
	}(ctx, cancel, composeApp)

	return nil, nil
}

func NewComposeService() *ComposeService {
	return &ComposeService{}
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
	if err := file.RMDir(workDir); err != nil {
		logger.Error("failed to cleanup working dir", zap.Error(err), zap.String("path", workDir))
	}
}
