package docker

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestIsImageStale_NegativeTest(t *testing.T) {
	stale, latestImage, err := IsImageStale("test", "123")
	assert.ErrorContains(t, err, "no such image")
	assert.Assert(t, !stale)
	assert.Equal(t, latestImage, "123")
}
