package service

import (
	"context"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
)

type ComposeService struct{}

func (s *ComposeService) Pull(ctx context.Context, composeApp *codegen.ComposeApp) error {
	panic("implement me")
}

func apiService() (*api.Service, error) {
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		return nil, err
	}

	dockerCli.Initialize(&flags.ClientOptions{
		Common: &flags.CommonOptions{},
	})

	apiService := compose.NewComposeService(dockerCli)
	return &apiService, nil
}
