package v2

import (
	"context"
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"gotest.tools/v3/assert"
)

func TestList(t *testing.T) {
	if !docker.IsDaemonRunning() {
		t.Skip("Docker daemon is not running")
	}

	service := NewComposeService()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := service.List(ctx)
	assert.NilError(t, err)
}
