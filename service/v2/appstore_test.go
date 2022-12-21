package v2

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestYAMLUnmarshal(t *testing.T) {
	for _, v := range catalog {
		assert.Equal(t, len(v.Project.Services), 1)
	}
}
