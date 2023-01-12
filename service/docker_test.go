package service

import (
	"context"
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/IceWhaleTech/CasaOS-Common/utils/random"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/samber/lo"
	"go.uber.org/goleak"
	"gotest.tools/v3/assert"
)

func TestRecreateContainer(t *testing.T) {
	defer goleak.VerifyNone(
		t,
		// https://github.com/docker/compose/issues/10157
		goleak.IgnoreTopFunction("github.com/docker/compose/v2/cmd/formatter.init.0.func1"),
	)

	if !docker.IsDaemonRunning() {
		t.Skip("Docker daemon is not running")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	assert.NilError(t, err)
	defer cli.Close()

	ctx := context.Background()

	// setup
	response := setupTestContainer(ctx, t)

	err = docker.StartContainer(ctx, response.ID)
	assert.NilError(t, err)

	// update
	newID, err := NewDockerService().RecreateContainer(ctx, response.ID, codegen.NotificationTypeNone)
	assert.NilError(t, err)

	defer func() {
		err := docker.RemoveContainer(ctx, newID)
		assert.NilError(t, err)
	}()

	defer func() {
		err := docker.StopContainer(ctx, newID)
		assert.NilError(t, err)
	}()

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	assert.NilError(t, err)

	assert.Assert(t, !lo.ContainsBy(containers, func(c types.Container) bool {
		return c.ID == response.ID
	}))
}

func setupTestContainer(ctx context.Context, t *testing.T) *container.CreateResponse {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	assert.NilError(t, err)
	defer cli.Close()

	config := &container.Config{
		Image: "alpine",
		Cmd:   []string{"tail", "-f", "/dev/null"},
	}

	hostConfig := &container.HostConfig{}
	networkingConfig := &network.NetworkingConfig{}

	response, err := cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, "test-"+random.RandomString(4, false))
	assert.NilError(t, err)

	return &response
}
