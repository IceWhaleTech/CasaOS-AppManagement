package v2

import (
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"

	"github.com/labstack/echo/v4"
)

func (a *AppManagement) GetGlobalSettings(ctx echo.Context) error {
	globalSettings := []codegen.GlobalSetting{
		// TODO: Add global settings here
	}

	return ctx.JSON(http.StatusOK, codegen.GlobalSettingsOK{
		Data: &globalSettings,
	})
}

func (a *AppManagement) GetGlobalSetting(ctx echo.Context, key codegen.GlobalSettingKey) error {
	return ctx.JSON(http.StatusOK, codegen.GlobalSettingOK{
		Data: &codegen.GlobalSetting{
			// TODO: Add global setting by key here
		},
	})
}

func (a *AppManagement) UpdateGlobalSetting(ctx echo.Context, key codegen.GlobalSettingKey) error {
	return ctx.JSON(http.StatusOK, codegen.GlobalSettingOK{
		Data: &codegen.GlobalSetting{
			// TODO: Add global setting by key here
		},
	})
}

func (a *AppManagement) DeleteGlobalSetting(ctx echo.Context, key codegen.GlobalSettingKey) error {
	return ctx.JSON(http.StatusOK, codegen.ResponseOK{})
}
