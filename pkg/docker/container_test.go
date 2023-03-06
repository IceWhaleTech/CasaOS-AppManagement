package docker_test

import (
	"context"
	"io"
	"runtime"
	"testing"

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

func setupTestContainer(ctx context.Context, t *testing.T) *container.CreateResponse {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	assert.NilError(t, err)
	defer cli.Close()

	imageName := "alpine:latest"

	config := &container.Config{
		Image: imageName,
		Cmd:   []string{"tail", "-f", "/dev/null"},
		Env:   []string{"FOO=BAR"},
	}

	hostConfig := &container.HostConfig{}
	networkingConfig := &network.NetworkingConfig{}

	out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	assert.NilError(t, err)

	_, err = io.ReadAll(out)
	assert.NilError(t, err)

	response, err := cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, "test-"+random.RandomString(4, false))
	assert.NilError(t, err)

	return &response
}

func TestCloneContainer(t *testing.T) {
	defer goleak.VerifyNone(t)

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	if !docker.IsDaemonRunning() {
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

	err = docker.StartContainer(ctx, response.ID)
	assert.NilError(t, err)

	defer func() {
		err = docker.StopContainer(ctx, response.ID)
		assert.NilError(t, err)
	}()

	newID, err := docker.CloneContainer(ctx, response.ID, "test-"+random.RandomString(4, false))
	assert.NilError(t, err)

	defer func() {
		err := docker.RemoveContainer(ctx, newID)
		assert.NilError(t, err)
	}()

	err = docker.StartContainer(ctx, newID)
	assert.NilError(t, err)

	defer func() {
		err := docker.StopContainer(ctx, newID)
		assert.NilError(t, err)
	}()

	containerInfo, err := docker.Container(ctx, newID)
	assert.NilError(t, err)
	assert.Assert(t, lo.Contains(containerInfo.Config.Env, "FOO=BAR"))
}

func TestNonExistingContainer(t *testing.T) {
	containerInfo, err := docker.Container(context.Background(), "non-existing-container")
	assert.ErrorContains(t, err, "non-existing-container")
	assert.Assert(t, containerInfo == nil)
}
