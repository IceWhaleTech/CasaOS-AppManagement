package docker

import (
	"context"
	"testing"

	"github.com/IceWhaleTech/CasaOS-Common/utils/random"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/samber/lo"
	"go.uber.org/goleak"
	"gotest.tools/v3/assert"
)

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

func TestRecreateContainer(t *testing.T) {
	defer goleak.VerifyNone(t)

	if !IsDaemonRunning() {
		t.Skip("Docker daemon is not running")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	assert.NilError(t, err)
	defer cli.Close()

	ctx := context.Background()

	// setup
	response := setupTestContainer(ctx, t)

	defer func() {
		err = cli.ContainerRemove(ctx, response.ID, types.ContainerRemoveOptions{})
		assert.NilError(t, err)
	}()

	err = StartContainer(ctx, response.ID)
	assert.NilError(t, err)

	defer func() {
		err = StopContainer(ctx, response.ID)
		assert.NilError(t, err)
	}()

	// recreate
	newID, err := RecreateContainer(ctx, response.ID, "test-"+random.RandomString(4, false))
	assert.NilError(t, err)

	defer func() {
		err := RemoveContainer(ctx, newID)
		assert.NilError(t, err)
	}()

	err = StartContainer(ctx, newID)
	assert.NilError(t, err)

	defer func() {
		err := StopContainer(ctx, newID)
		assert.NilError(t, err)
	}()
}

func TestUpdateContainerWithNewImage(t *testing.T) {
	defer goleak.VerifyNone(t)

	if !IsDaemonRunning() {
		t.Skip("Docker daemon is not running")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	assert.NilError(t, err)
	defer cli.Close()

	ctx := context.Background()

	// setup
	response := setupTestContainer(ctx, t)

	err = StartContainer(ctx, response.ID)
	assert.NilError(t, err)

	// update
	err = UpdateContainerWithNewImage(ctx, response.ID, true)
	assert.NilError(t, err)

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	assert.NilError(t, err)

	assert.Assert(t, !lo.ContainsBy(containers, func(c types.Container) bool {
		return c.ID == response.ID
	}))
}
