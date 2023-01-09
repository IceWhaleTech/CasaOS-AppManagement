package v2

import (
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/labstack/echo/v4"
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

func (a *AppManagement) RecreateContainerByID(ctx echo.Context, id codegen.ContainerID) error {
	panic("implement me")
}
