package v2

import (
	"context"
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gotest.tools/v3/assert/cmp"
)

func (a *AppManagement) CheckContainerHealthByID(ctx echo.Context, id codegen.ContainerID) error {
	result, err := service.MyService.Docker().CheckContainerHealth(id)
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusServiceUnavailable, codegen.ResponseServiceUnavailable{Message: &message})
	}

	if !result {
		return ctx.JSON(http.StatusServiceUnavailable, codegen.ResponseServiceUnavailable{})
	}

	return ctx.JSON(http.StatusOK, codegen.ContainerHealthCheckOK{})
}

func (a *AppManagement) RecreateContainerByID(ctx echo.Context, id codegen.ContainerID, params codegen.RecreateContainerByIDParams) error {
	// attach context key/value pairs from upstream
	backgroundCtx := common.WithProperties(context.Background(), PropertiesFromQueryParams(ctx))

	if _, err := service.MyService.Docker().DescribeContainer(backgroundCtx, id); err != nil {
		message := err.Error()

		if cmp.ErrorContains(err, "non-existing-container")().Success() {
			return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{Message: &message})
		}

		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseNotFound{Message: &message})
	}

	pullLatestImage := false
	if params.Pull != nil {
		pullLatestImage = *params.Pull
	}

	go func() {
		if err := service.MyService.Docker().RecreateContainer(backgroundCtx, id, pullLatestImage); err != nil {
			logger.Error("error when trying to recreate container", zap.Error(err), zap.String("containerID", string(id)), zap.Bool("pull", pullLatestImage))
		}
	}()

	return ctx.JSON(http.StatusOK, codegen.ContainerRecreateOK{
		Message: utils.Ptr("Container is being recreated asynchronously"),
	})
}
