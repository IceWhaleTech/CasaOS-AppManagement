package docker

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestCurrentArchitecture(t *testing.T) {
	a, err := CurrentArchitecture()
	assert.NilError(t, err)
	assert.Assert(t, a != "")
}
