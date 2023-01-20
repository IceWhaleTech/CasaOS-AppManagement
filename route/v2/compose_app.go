package v2

import (
	"errors"
	"io"
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var ErrComposeAppNameNotProvided = errors.New("compose app name is not provided")

func (a *AppManagement) MyComposeAppList(ctx echo.Context) error {
	composeApps, err := service.MyService.Compose().List(ctx.Request().Context())
	if err != nil {
		message := err.Error()
		logger.Error("failed to list compose apps", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	return ctx.JSON(http.StatusOK, composeApps)
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
