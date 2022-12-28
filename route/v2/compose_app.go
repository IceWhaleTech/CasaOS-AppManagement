package v2

import (
	"fmt"
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	v2 "github.com/IceWhaleTech/CasaOS-AppManagement/service/v2"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

func (a *AppManagement) MyComposeAppList(ctx echo.Context) error {
	panic("implement me")
}

func (a *AppManagement) InstallComposeApp(ctx echo.Context) error {
	var composeApp *v2.ComposeApp

	switch ctx.Request().Header.Get(echo.HeaderAccept) {
	case MIMEApplicationYAML:

		reader, err := ctx.Request().GetBody()
		if err != nil {
			message := err.Error()
			logger.Error("failed to get body from the request", zap.Error(err))
			return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
		}

		len := lo.If(ctx.Request().ContentLength > 0, ctx.Request().ContentLength+1).Else(16 * 1024) // default: 16k
		buf := make([]byte, len)
		n, err := reader.Read(buf)
		if err != nil {
			message := err.Error()
			logger.Error("failed to read body from the request", zap.Error(err))
			return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
		}

		composeApp, err = v2.NewComposeAppFromYAML(buf[:n])
		if err != nil {
			message := err.Error()
			logger.Error("failed to load compose app", zap.Error(err))
			return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
		}

	case echo.MIMEApplicationJSON:
		var c codegen.ComposeApp
		if err := ctx.Bind(&c); err != nil {
			message := err.Error()
			logger.Error("failed to decode JSON from the request", zap.Error(err))
			return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{
				Message: &message,
			})
		}

		composeApp = (*v2.ComposeApp)(&c)

	default:
		return ctx.JSON(http.StatusNotAcceptable, codegen.ResponseBadRequest{
			Message: utils.Ptr(fmt.Sprintf("Content is not acceptable - `Accept` in HTTP headers must be either %s or %s", MIMEApplicationYAML, echo.MIMEApplicationJSON)),
		})
	}

	composeApp, err := composeApp.Install()
	if err != nil {
		message := err.Error()
		logger.Error("failed to install compose app", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	return ctx.JSON(http.StatusOK, composeApp)
}
