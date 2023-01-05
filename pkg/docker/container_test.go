package docker

import (
	"context"
	"testing"

	"github.com/IceWhaleTech/CasaOS-Common/utils/random"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"gotest.tools/v3/assert"
)

func TestRecreateContainer(t *testing.T) {
	if !IsDaemonRunning() {
		t.Skip("Docker daemon is not running")
	}

	// setup
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	assert.NilError(t, err)
	defer cli.Close()

	ctx := context.Background()

	config := &container.Config{
		Image: "alpine",
		Cmd:   []string{"tail", "-f", "/dev/null"},
	}

	hostConfig := &container.HostConfig{}
	networkingConfig := &network.NetworkingConfig{}

	response, err := cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, "test-"+random.RandomString(4, false))
	assert.NilError(t, err)

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
