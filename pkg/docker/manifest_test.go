package docker

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
)

func TestRemoteManife(t *testing.T) {
	ctx := context.Background()
	manifest, err := RemoteManifest(ctx, "hello-world:latest")
	assert.NilError(t, err)
	assert.Assert(t, manifest != nil)
}
