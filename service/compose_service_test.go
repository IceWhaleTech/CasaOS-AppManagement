package service

import (
	"context"
	"testing"

	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"go.uber.org/goleak"
	"gotest.tools/v3/assert"
)

func TestList(t *testing.T) {
	defer goleak.VerifyNone(t)

	if !docker.IsDaemonRunning() {
		t.Skip("Docker daemon is not running")
	}

	logger.LogInitConsoleOnly()

	service := NewComposeService()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := service.List(ctx)
	assert.NilError(t, err)
}
