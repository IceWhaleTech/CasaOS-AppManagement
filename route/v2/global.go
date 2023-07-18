package v2

import (
	"context"
	"fmt"
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"go.uber.org/zap"

	"github.com/labstack/echo/v4"
)

func (a *AppManagement) GetGlobalSettings(ctx echo.Context) error {
	result := make([]codegen.GlobalSetting, 0)
	for key, value := range config.Global {
		result = append(result, codegen.GlobalSetting{
			Key:         utils.Ptr(key),
			Value:       value,
			Description: utils.Ptr(key),
		})
	}

	return ctx.JSON(http.StatusOK, codegen.GlobalSettingsOK{
		Data: &result,
	})
}

func (a *AppManagement) GetGlobalSetting(ctx echo.Context, key codegen.GlobalSettingKey) error {
	value, ok := config.Global[string(key)]
	if !ok {
		message := "the key is not exist"
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.GlobalSettingOK{
		Data: &codegen.GlobalSetting{
			Key:         utils.Ptr(key),
			Value:       value,
			Description: utils.Ptr(key),
		},
	})
}

func (a *AppManagement) UpdateGlobalSetting(ctx echo.Context, key codegen.GlobalSettingKey) error {
	var action codegen.GlobalSetting
	if err := ctx.Bind(&action); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}
	if err := updateGlobalEnv(ctx, key, action.Value); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.GlobalSettingOK{
		Data: &codegen.GlobalSetting{
			Key:         utils.Ptr(key),
			Value:       action.Value,
			Description: utils.Ptr(key),
		},
	})
}

func updateGlobalEnv(ctx echo.Context, key string, value string) error {
	if key == "" {
		return fmt.Errorf("openai api key is required")
	}
	if err := service.MyService.AppStoreManagement().ChangeGlobal(key, value); err != nil {
		return err
	}

	// re up all containers to apply the new env var
	go func() {
		backgroundCtx := common.WithProperties(context.Background(), PropertiesFromQueryParams(ctx))
		composeAppsWithStoreInfo, err := service.MyService.Compose().List(backgroundCtx)
		if err != nil {
			logger.Error("Failed to get composeAppsWithStoreInfo", zap.Any("error", err))
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

func deleteGlobalEnv(ctx echo.Context, key string) error {

	if err := service.MyService.AppStoreManagement().DeleteGlobal(key); err != nil {
		return err
	}

	// re up all containers to apply the new env var
	go func() {
		backgroundCtx := common.WithProperties(context.Background(), PropertiesFromQueryParams(ctx))
		composeAppsWithStoreInfo, err := service.MyService.Compose().List(backgroundCtx)
		if err != nil {
			logger.Error("Failed to get composeAppsWithStoreInfo", zap.Any("error", err))
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
	var action codegen.GlobalSetting
	if err := ctx.Bind(&action); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	if err := deleteGlobalEnv(ctx, key); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.ResponseOK{})
}
