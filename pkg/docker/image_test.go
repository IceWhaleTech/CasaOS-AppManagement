package docker

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestIsImageStale_NoSuchImage(t *testing.T) {
	stale, latestImage, err := IsImageStale("test", "123")
	assert.ErrorContains(t, err, "no such image")
	assert.Assert(t, !stale)
	assert.Equal(t, latestImage, "123")
}

func TestIsImageStale(t *testing.T) {
	imageName := "hello-world"

	err := PullImage(imageName, nil)
	assert.NilError(t, err)

	stale1, latestImage1, err1 := IsImageStale(imageName, "123")
	assert.NilError(t, err1)
	assert.Assert(t, stale1)
	assert.Assert(t, latestImage1 != "123")

	stale2, latestImage2, err2 := IsImageStale(imageName, latestImage1)
	assert.NilError(t, err2)
	assert.Assert(t, !stale2)
	assert.Equal(t, latestImage2, latestImage1)
}
