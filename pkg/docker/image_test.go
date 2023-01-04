package docker

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
)

func TestIsImageStale_NoSuchImage(t *testing.T) {
	imageName := "test"

	ctx := context.Background()

	err := PullNewImage(ctx, imageName)
	assert.ErrorContains(t, err, "no such image")

	stale, latestImage, _ := HasNewImage(ctx, imageName, "123")
	assert.Assert(t, !stale)
	assert.Equal(t, latestImage, "123")
}

func TestIsImageStale(t *testing.T) {
	imageName := "hello-world"

	ctx := context.Background()

	err := PullNewImage(ctx, imageName)
	assert.NilError(t, err)

	stale1, latestImage1, err1 := HasNewImage(ctx, imageName, "123")
	assert.NilError(t, err1)
	assert.Assert(t, stale1)
	assert.Assert(t, latestImage1 != "123")

	stale2, latestImage2, err2 := HasNewImage(ctx, imageName, latestImage1)
	assert.NilError(t, err2)
	assert.Assert(t, !stale2)
	assert.Equal(t, latestImage2, latestImage1)
}
