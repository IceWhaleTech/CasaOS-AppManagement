package docker

import (
	"context"
	"io"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"go.uber.org/goleak"
	"gotest.tools/v3/assert"
)

func TestCompareDigest(t *testing.T) {
	defer goleak.VerifyNone(t)

	if !IsDaemonRunning() {
		t.Skip("Docker daemon is not running")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	assert.NilError(t, err)
	defer cli.Close()

	ctx := context.Background()

	imageName := "alpine:latest"

	out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	assert.NilError(t, err)
	defer out.Close()

	str, err := io.ReadAll(out)
	assert.NilError(t, err)

	t.Log(string(str))

	imageInfo, _, err := cli.ImageInspectWithRaw(ctx, imageName)
	assert.NilError(t, err)

	match, err := CompareDigest(imageName, imageInfo.RepoDigests, "")
	assert.NilError(t, err)

	assert.Assert(t, match)
}
