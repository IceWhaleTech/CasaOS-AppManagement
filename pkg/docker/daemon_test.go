package docker_test

import (
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/docker/docker/client"
	"gotest.tools/v3/assert"
)

func TestCurrentArchitecture(t *testing.T) {
	a, err := docker.CurrentArchitecture()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	assert.NilError(t, err)
	defer cli.Close()

	assert.NilError(t, err)
	assert.Assert(t, a != "")
}

func TestCurrentVersion(t *testing.T) {
	v, err := docker.CurrentVersion()
	assert.NilError(t, err)
	assert.Assert(t, v != "")
}
