package docker

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestIsImageStale_NegativeTest(t *testing.T) {
	stale, latestImage, err := IsImageStale("test", "123")
	assert.Error(t, err, "container uses a pinned image, and cannot be updated")

	assert.Assert(t, !stale)
	assert.Equal(t, latestImage, "123")
}
