package v2

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestMarshalJSON(t *testing.T) {
	a, err := NewComposeAppFromYAML([]byte(SampleComposeAppYAML), nil)
	assert.NilError(t, err)

	buf, err := json.Marshal(a)
	assert.NilError(t, err)

	var b ComposeApp
	err = json.Unmarshal(buf, &b)
	assert.NilError(t, err)
}
