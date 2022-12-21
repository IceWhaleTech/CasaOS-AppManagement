package v2

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestGetComposeApp(t *testing.T) {
	for storeAppID, composeApp := range Store {
		storeInfo, err := composeApp.StoreInfo()
		assert.NilError(t, err)
		assert.Equal(t, storeInfo.StoreAppID, storeAppID)
	}
}

func TestComposeYAML(t *testing.T) {
	for _, composeApp := range Store {
		assert.Equal(t, *composeApp.YAML(), SampleComposeAppYAML)
	}
}
