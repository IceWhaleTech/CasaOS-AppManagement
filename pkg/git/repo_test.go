package git

import (
	"os"
	"testing"

	"go.uber.org/goleak"
	"gotest.tools/v3/assert"
)

const gitURL = "https://github.com/IceWhaleTech/get"

func TestValidateGitURL(t *testing.T) {
	goleak.VerifyNone(t)

	err := ValidateGitURL(gitURL)
	assert.NilError(t, err)

	err = ValidateGitURL("https://github.com/Foo/Bar")
	assert.Assert(t, err != nil)

	err = ValidateGitURL("invalid url")
	assert.Assert(t, err != nil)
}

func TestWorkDir(t *testing.T) {
	goleak.VerifyNone(t)

	dir, err := WorkDir(gitURL, "/tmp")
	assert.NilError(t, err)
	assert.Equal(t, dir, "/tmp/github.com/icewhaletech/get")
}

func TestCloneAndPull(t *testing.T) {
	goleak.VerifyNone(t)

	dir, err := WorkDir(gitURL, "/tmp")
	assert.NilError(t, err)
	defer os.RemoveAll(dir)

	err = Clone(gitURL, dir)
	assert.NilError(t, err)

	err = Pull(dir)
	assert.NilError(t, err)
}
