package docker

import (
	"context"
	"io"
	"testing"

	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/mitchellh/mapstructure"
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

	match, err := CompareDigest(imageName, imageInfo.RepoDigests)
	assert.NilError(t, err)

	assert.Assert(t, match)
}

func TestGetManifest1(t *testing.T) {
	defer goleak.VerifyNone(t)

	manifest, contentType, err := GetManifest(context.Background(), "hello-world:latest")
	assert.NilError(t, err)
	assert.Equal(t, contentType, manifestlist.MediaTypeManifestList)

	var listManifest manifestlist.ManifestList
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{Result: &listManifest, Squash: true})
	assert.NilError(t, err)

	err = decoder.Decode(manifest)
	assert.NilError(t, err)
	assert.Assert(t, len(listManifest.Manifests) > 0)
}

func TestGetManifest2(t *testing.T) {
	defer goleak.VerifyNone(t)
	manifest, contentType, err := GetManifest(context.Background(), "wangxiaohu/brother-cups:latest")
	assert.NilError(t, err)
	assert.Equal(t, contentType, schema1.MediaTypeSignedManifest)

	var signedManifest schema1.SignedManifest
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{Result: &signedManifest, Squash: true})
	assert.NilError(t, err)

	err = decoder.Decode(manifest)
	assert.NilError(t, err)
	assert.Assert(t, len(signedManifest.Architecture) > 0)
}
