package docker_test

import (
	"context"
	"io"
	"runtime"
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
	"go.uber.org/goleak"
	"gotest.tools/v3/assert"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestCompareDigest(t *testing.T) {
	defer goleak.VerifyNone(t)

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	if !docker.IsDaemonRunning() {
		t.Skip("Docker daemon is not running")
	}

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

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

	match, err := docker.CompareDigest(imageName, imageInfo.RepoDigests)
	assert.NilError(t, err)

	assert.Assert(t, match)
}

func TestGetManifest1(t *testing.T) {
	defer goleak.VerifyNone(t)

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	manifest, contentType, err := docker.GetManifest(context.Background(), "hello-world:nanoserver-1803")
	assert.NilError(t, err)
	assert.Equal(t, contentType, manifestlist.MediaTypeManifestList)

	var listManifest manifestlist.ManifestList
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{Result: &listManifest, Squash: true})
	assert.NilError(t, err)

	err = decoder.Decode(manifest)
	assert.NilError(t, err)

	architectures := lo.Map(listManifest.Manifests, func(m manifestlist.ManifestDescriptor, i int) string {
		return m.Platform.Architecture
	})

	architectures = lo.Filter(architectures, func(a string, i int) bool {
		return a != ""
	})

	assert.Assert(t, len(architectures) > 0)
}

func TestGetManifest2(t *testing.T) {
	defer goleak.VerifyNone(t)

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	manifest, contentType, err := docker.GetManifest(context.Background(), "correctroad/logseq:latest")
	assert.NilError(t, err)
	assert.Equal(t, contentType, "application/vnd.docker.distribution.manifest.v2+json")

	var signedManifest schema1.SignedManifest
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{Result: &signedManifest, Squash: true})
	assert.NilError(t, err)

	err = decoder.Decode(manifest)
	assert.NilError(t, err)

}

func TestGetManifest3(t *testing.T) {
	defer goleak.VerifyNone(t)

	defer func() {
		// workaround due to https://github.com/patrickmn/go-cache/issues/166
		docker.Cache = nil
		runtime.GC()
	}()

	manifest, contentType, err := docker.GetManifest(context.Background(), "2fauth/2fauth:latest")
	assert.NilError(t, err)
	assert.Equal(t, contentType, v1.MediaTypeImageIndex)

	var listManifest manifestlist.ManifestList
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{Result: &listManifest, Squash: true})
	assert.NilError(t, err)

	err = decoder.Decode(manifest)
	assert.NilError(t, err)

	architectures := lo.Map(listManifest.Manifests, func(m manifestlist.ManifestDescriptor, i int) string {
		return m.Platform.Architecture
	})

	architectures = lo.Uniq(architectures)

	architectures = lo.Filter(architectures, func(a string, i int) bool {
		return a != "unknown" && a != ""
	})

	assert.Assert(t, len(architectures) > 0)
}
