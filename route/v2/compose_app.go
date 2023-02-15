package v2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	v2 "github.com/IceWhaleTech/CasaOS-AppManagement/service/v2"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/compose-spec/compose-go/types"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var ErrComposeAppIDNotProvided = errors.New("compose AppID (compose project name) is not provided")

func (a *AppManagement) MyComposeAppList(ctx echo.Context) error {
	composeApps, err := service.MyService.Compose().List(ctx.Request().Context())
	if err != nil {
		message := err.Error()
		logger.Error("failed to list compose apps", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	composeAppsWithStoreInfo := lo.MapValues(composeApps, func(composeApp *v2.ComposeApp, id string) codegen.ComposeAppWithStoreInfo {
		storeInfo, err := composeApp.StoreInfo(true)
		if err != nil {
			logger.Error("failed to get store info", zap.Error(err), zap.String("composeAppID", id))
		}

		return codegen.ComposeAppWithStoreInfo{
			Compose:   (*codegen.ComposeApp)(composeApp),
			StoreInfo: storeInfo,
		}
	})

	return ctx.JSON(http.StatusOK, codegen.ComposeAppListOK{
		Data: &composeAppsWithStoreInfo,
	})
}

func (a *AppManagement) MyComposeApp(ctx echo.Context, id codegen.ComposeAppID) error {
	if id == "" {
		message := ErrComposeAppIDNotProvided.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{
			Message: &message,
		})
	}

	composeApps, err := service.MyService.Compose().List(ctx.Request().Context())
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	composeApp, ok := composeApps[id]
	if !ok {
		message := fmt.Sprintf("compose app `%s` not found", id)
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{Message: &message})
	}

	accept := ctx.Request().Header.Get(echo.HeaderAccept)
	if accept == common.MIMEApplicationYAML {
		yaml, err := yaml.Marshal(composeApp)
		if err != nil {
			message := err.Error()
			return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{
				Message: &message,
			})
		}

		return ctx.String(http.StatusOK, string(yaml))
	}

	storeInfo, err := composeApp.StoreInfo(true)
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{
			Message: &message,
		})
	}

	message := fmt.Sprintf("!! JSON format is for debugging purpose only - use `Accept: %s` HTTP header to get YAML instead !!", common.MIMEApplicationYAML)
	return ctx.JSON(http.StatusOK, codegen.ComposeAppOK{
		// extension properties aren't marshalled - https://github.com/golang/go/issues/6213
		Message: &message,
		Data: &codegen.ComposeAppWithStoreInfo{
			StoreInfo: storeInfo,
			Compose:   (*types.Project)(composeApp),
		},
	})
}

func (a *AppManagement) UpdateComposeAppSettings(ctx echo.Context, id codegen.ComposeAppID) error {
	if id == "" {
		message := ErrComposeAppIDNotProvided.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{
			Message: &message,
		})
	}

	composeApps, err := service.MyService.Compose().List(ctx.Request().Context())
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	composeApp, ok := composeApps[id]
	if !ok {
		message := fmt.Sprintf("compose app `%s` not found", id)
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{Message: &message})
	}

	buf, err := YAMLfromRequest(ctx)
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{
			Message: &message,
		})
	}

	// attach context key/value pairs from upstream
	backgroundCtx := common.WithProperties(context.Background(), PropertiesFromQueryParams(ctx))

	if err := composeApp.UpdateSettings(backgroundCtx, buf); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{
			Message: &message,
		})
	}

	return ctx.JSON(http.StatusOK, codegen.ComposeAppUpdateSettingsOK{
		Message: utils.Ptr("compose app settings are updated"),
	})
}

func (a *AppManagement) InstallComposeApp(ctx echo.Context) error {
	buf, err := YAMLfromRequest(ctx)
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{
			Message: &message,
		})
	}

	// attach context key/value pairs from upstream
	backgroundCtx := common.WithProperties(context.Background(), PropertiesFromQueryParams(ctx))

	if err := service.MyService.Compose().Install(backgroundCtx, buf); err != nil {
		message := err.Error()

		if err == v2.ErrComposeExtensionNameXCasaOSNotFound {
			return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
		}

		logger.Error("failed to start compose app installation", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.ComposeAppInstallOK{
		Message: utils.Ptr("compose app is being installed asynchronously"),
	})
}

func (a *AppManagement) UninstallComposeApp(ctx echo.Context, id codegen.ComposeAppID) error {
	if id == "" {
		message := ErrComposeAppIDNotProvided.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{
			Message: &message,
		})
	}

	// attach context key/value pairs from upstream
	backgroundCtx := common.WithProperties(context.Background(), PropertiesFromQueryParams(ctx))

	if err := service.MyService.Compose().Uninstall(backgroundCtx, id); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.ComposeAppUninstallOK{})
}

func (a *AppManagement) ComposeAppStatus(ctx echo.Context, id codegen.ComposeAppID) error {
	if id == "" {
		message := ErrComposeAppIDNotProvided.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{
			Message: &message,
		})
	}

	statusList, err := service.MyService.Compose().StatusList(ctx.Request().Context())
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	status, ok := statusList[id]
	if !ok {
		message := fmt.Sprintf("compose app `%s` not found", id)
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.ComposeAppStatusOK{
		Data: &status,
	})
}

func (a *AppManagement) SetComposeAppStatus(ctx echo.Context, id codegen.ComposeAppID) error {
	if id == "" {
		message := ErrComposeAppIDNotProvided.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{
			Message: &message,
		})
	}

	var action codegen.RequestComposeAppStatus
	if err := ctx.Bind(&action); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	switch action {
	case codegen.RequestComposeAppStatusStart:
	case codegen.RequestComposeAppStatusStop:
	case codegen.RequestComposeAppStatusRestart:
	default:
		message := fmt.Sprintf("invalid action `%s`", action)
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}
	return ctx.JSON(http.StatusOK, codegen.RequestComposeAppStatusOK{})
}

func (a *AppManagement) ComposeAppLogs(ctx echo.Context, id codegen.ComposeAppID, params codegen.ComposeAppLogsParams) error {
	if id == "" {
		message := ErrComposeAppIDNotProvided.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{
			Message: &message,
		})
	}

	composeApps, err := service.MyService.Compose().List(ctx.Request().Context())
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	composeApp, ok := composeApps[id]
	if !ok {
		message := fmt.Sprintf("compose app `%s` not found", id)
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{Message: &message})
	}

	lines := lo.If(params.Lines == nil, 1000).Else(*params.Lines)
	logs, err := composeApp.Logs(ctx.Request().Context(), lines)
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.ComposeAppLogsOK{Data: utils.Ptr(string(logs))})
}

func (a *AppManagement) ComposeAppContainers(ctx echo.Context, id codegen.ComposeAppID) error {
	if id == "" {
		message := ErrComposeAppIDNotProvided.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{
			Message: &message,
		})
	}

	composeApps, err := service.MyService.Compose().List(ctx.Request().Context())
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	composeApp, ok := composeApps[id]
	if !ok {
		message := fmt.Sprintf("compose app `%s` not found", id)
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{Message: &message})
	}

	containers, err := composeApp.Containers(ctx.Request().Context())
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	storeInfo, err := composeApp.StoreInfo(false)
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.ComposeAppContainersOK{
		Data: &codegen.ComposeAppContainers{
			Main:       storeInfo.MainApp,
			Containers: &containers,
		},
	})
}

func YAMLfromRequest(ctx echo.Context) ([]byte, error) {
	var buf []byte

	switch ctx.Request().Header.Get(echo.HeaderContentType) {
	case common.MIMEApplicationYAML:

		_buf, err := io.ReadAll(ctx.Request().Body)
		if err != nil {
			return nil, err
		}

		buf = _buf

	default:
		var c codegen.ComposeApp
		if err := ctx.Bind(&c); err != nil {
			return nil, err
		}

		_buf, err := yaml.Marshal(c)
		if err != nil {
			return nil, err
		}
		buf = _buf
	}

	return buf, nil
}
