package v2

import (
	"testing"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"gotest.tools/v3/assert"
)

func TestYAMLUnmarshal(t *testing.T) {
	for _, v := range catalog {
		project, err := loader.Load(types.ConfigDetails{
			ConfigFiles: []types.ConfigFile{
				{
					Content: []byte(v),
				},
			},
		})

		assert.NilError(t, err)
		assert.Equal(t, len(project.Services), 1)
	}
}
