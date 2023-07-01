package v2

import (
	"context"
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"go.uber.org/zap"

	"github.com/labstack/echo/v4"
)

// temp
var (
	OPENAPI_AI_KEY string = "OPENAPI_AI_KEY"
)

// to how process single and all relation?
func (a *AppManagement) getGlobalSettings() *[]codegen.GlobalSetting {
	return &[]codegen.GlobalSetting{
		codegen.GlobalSetting{
			Key:         &OPENAPI_AI_KEY,
			Value:       config.AppInfo.OpenAIAPIKey,
			Description: &OPENAPI_AI_KEY,
		},
	}
}

func (a *AppManagement) GetGlobalSettings(ctx echo.Context) error {
	// TODO: implement logic to read [global] settings from conf file

	return ctx.JSON(http.StatusOK, codegen.GlobalSettingsOK{
		Data: a.getGlobalSettings(),
	})
}

func (a *AppManagement) GetGlobalSetting(ctx echo.Context, key codegen.GlobalSettingKey) error {
	var result codegen.GlobalSetting

	switch key {
	case "OPENAPI_AI_KEY":
		result = codegen.GlobalSetting{
			Key:         &OPENAPI_AI_KEY,
			Value:       config.AppInfo.OpenAIAPIKey,
			Description: &OPENAPI_AI_KEY,
		}
	}

	// TODO: implement logic to read a specific [global] setting from conf file
	return ctx.JSON(http.StatusOK, codegen.GlobalSettingOK{
		Data: &result,
	})
}

func (a *AppManagement) UpdateGlobalSetting(ctx echo.Context, key codegen.GlobalSettingKey) error {
	// TODO: implement logic to update a specific [global] setting in conf file, and cache in memory
	var action codegen.GlobalSetting
	if err := ctx.Bind(&action); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	switch key {
	case "OPENAPI_AI_KEY":
		if err := updateOpenAIAPIKey(ctx, action.Value); err != nil {
			message := "openai api key is required"
			return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
		}
	}

	return ctx.JSON(http.StatusOK, codegen.GlobalSettingOK{
		Data: &codegen.GlobalSetting{
			// TODO: Add global setting by key here
		},
	})

}

func updateOpenAIAPIKey(ctx echo.Context, key string) error {
	if key == "" {
		message := "openai api key is required"
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	if err := service.MyService.AppStoreManagement().ChangeOpenAIAPIKey(key); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	// re up all containers
	go func() {
		backgroundCtx := common.WithProperties(context.Background(), PropertiesFromQueryParams(ctx))
		composeAppsWithStoreInfo, err := service.MyService.Compose().List(backgroundCtx)
		if err != nil {

		}
		for _, project := range composeAppsWithStoreInfo {
			if service, _, err := service.ApiService(); err == nil {
				project.UpWithCheckRequire(backgroundCtx, service)
			} else {
				logger.Error("Failed to get Api Service", zap.Any("error", err))
			}
		}
	}()

	return nil
}

func (a *AppManagement) DeleteGlobalSetting(ctx echo.Context, key codegen.GlobalSettingKey) error {
	// TODO: implement logic to delete a specific [global] setting in conf file, and remove from cache in memory

	return ctx.JSON(http.StatusOK, codegen.ResponseOK{})
}
