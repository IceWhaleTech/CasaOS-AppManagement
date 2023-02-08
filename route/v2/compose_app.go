package v2

import (
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

	return ctx.JSON(http.StatusOK, composeApps)
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
		yaml, err := composeApp.YAML()
		if err != nil {
			message := err.Error()
			return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{
				Message: &message,
			})
		}

		return ctx.String(http.StatusOK, *yaml)
	}

	storeInfo, err := composeApp.StoreInfo()
	if err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{
			Message: &message,
		})
	}

	return ctx.JSON(http.StatusOK, codegen.ComposeAppOK{
		// extension properties aren't marshalled - https://github.com/golang/go/issues/6213
		Data: &codegen.ComposeAppWithStoreInfo{
			StoreInfo: storeInfo,
			Compose:   (*types.Project)(composeApp),
		},
	})
}

func (a *AppManagement) InstallComposeApp(ctx echo.Context) error {
	var buf []byte

	switch ctx.Request().Header.Get(echo.HeaderContentType) {
	case common.MIMEApplicationYAML:

		_buf, err := io.ReadAll(ctx.Request().Body)
		if err != nil {
			message := err.Error()
			logger.Error("failed to read body from the request", zap.Error(err))
			return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
		}

		buf = _buf

	default:
		var c codegen.ComposeApp
		if err := ctx.Bind(&c); err != nil {
			message := err.Error()
			logger.Error("failed to decode JSON from the request", zap.Error(err))
			return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{
				Message: &message,
			})
		}

		_buf, err := yaml.Marshal(c)
		if err != nil {
			message := err.Error()
			logger.Error("failed to marshal compose app", zap.Error(err))
			return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
		}
		buf = _buf
	}

	if err := service.MyService.Compose().Install(buf); err != nil {
		message := err.Error()
		logger.Error("failed to start compose app installation", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.ComposeAppInstallOK{
		Message: utils.Ptr("compose app is being installed asynchronously"),
	})
}

func (a *AppManagement) ComposeAppStatus(ctx echo.Context, id codegen.ComposeAppID) error {
	if id == "" {
		message := ErrComposeAppIDNotProvided.Error()
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{
			Message: &message,
		})
	}

	status, err := service.MyService.Compose().Status(ctx.Request().Context(), id)
	if err != nil {
		message := err.Error()

		if err == v2.ErrComposeAppNotFound {
			return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{Message: &message})
		}

		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.ComposeAppStatusOK{
		Data: &status,
	})
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

	storeInfo, err := composeApp.StoreInfo()
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
