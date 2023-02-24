package git

import (
	"testing"

	"go.uber.org/goleak"
	"gotest.tools/v3/assert"
)

func TestValidateGitURL(t *testing.T) {
	goleak.VerifyNone(t)

	gitURL := "https://github.com/IceWhaleTech/CasaOS-AppStore"

	err := ValidateGitURL(gitURL)
	assert.NilError(t, err)

	gitURL = "https://github.com/Foo/Bar"

	err = ValidateGitURL(gitURL)
	assert.Assert(t, err != nil)

	gitURL = "invalid url"
	err = ValidateGitURL(gitURL)
	assert.Assert(t, err != nil)
}
