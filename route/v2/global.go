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

func getGlobalSettingsKeyAndValue() map[string]map[string]string {
	return map[string]map[string]string{
		"OPENAPI_AI_KEY": map[string]string{
			"key":         "OPENAPI_AI_KEY",
			"value":       config.AppInfo.OpenAIAPIKey,
			"description": "OPENAPI_AI_KEY",
		},
	}
}

func (a *AppManagement) GetGlobalSettings(ctx echo.Context) error {
	result := make([]codegen.GlobalSetting, 0)
	for _, v := range getGlobalSettingsKeyAndValue() {
		result = append(result, codegen.GlobalSetting{
			Key:         utils.Ptr(v["key"]),
			Value:       v["value"],
			Description: utils.Ptr(v["description"]),
		})
	}

	return ctx.JSON(http.StatusOK, codegen.GlobalSettingsOK{
		Data: &result,
	})
}

func (a *AppManagement) GetGlobalSetting(ctx echo.Context, key codegen.GlobalSettingKey) error {
	if _, ok := getGlobalSettingsKeyAndValue()[string(key)]; !ok {
		message := "the key is not exist"
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.GlobalSettingOK{
		Data: &codegen.GlobalSetting{
			Key:         utils.Ptr(getGlobalSettingsKeyAndValue()[string(key)]["key"]),
			Value:       getGlobalSettingsKeyAndValue()[string(key)]["value"],
			Description: utils.Ptr(getGlobalSettingsKeyAndValue()[string(key)]["description"]),
		},
	})
}

func (a *AppManagement) UpdateGlobalSetting(ctx echo.Context, key codegen.GlobalSettingKey) error {
	var action codegen.GlobalSetting
	if err := ctx.Bind(&action); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	if _, ok := getGlobalSettingsKeyAndValue()[string(key)]; !ok {
		message := "the key is not exist"
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{Message: &message})
	}

	switch key {
	case "OPENAPI_AI_KEY":
		if err := updateOpenAIAPIKey(ctx, action.Value); err != nil {
			message := err.Error()
			return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
		}
	}

	return ctx.JSON(http.StatusOK, codegen.GlobalSettingOK{
		Data: &codegen.GlobalSetting{
			Key:         utils.Ptr(getGlobalSettingsKeyAndValue()[string(key)]["key"]),
			Value:       getGlobalSettingsKeyAndValue()[string(key)]["value"],
			Description: utils.Ptr(getGlobalSettingsKeyAndValue()[string(key)]["description"]),
		},
	})
}

func updateOpenAIAPIKey(ctx echo.Context, key string) error {
	if key == "" {
		return fmt.Errorf("openai api key is required")
	}

	if err := service.MyService.AppStoreManagement().ChangeOpenAIAPIKey(key); err != nil {
		return err
	}

	// re up all containers to apply the new env var
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
