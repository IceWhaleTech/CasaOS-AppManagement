package downloadHelper

import (
	"testing"

	"go.uber.org/goleak"
	"gotest.tools/v3/assert"
)

func TestDownload(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start")) // https://github.com/census-instrumentation/opencensus-go/issues/1191

	src := "https://github.com/IceWhaleTech/get/archive/refs/heads/main.zip"

	dst := t.TempDir()

	err := Download(src, dst)
	assert.NilError(t, err)
}
