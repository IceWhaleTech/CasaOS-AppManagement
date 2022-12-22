package v2

import (
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/labstack/echo/v4"
)

func (*AppManagement) GetAppInfo(ctx echo.Context, id codegen.AppStoreID) error {
	composeApp := service.MyService.V2AppStore().ComposeApp(id)

	if composeApp == nil {
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{
			Message: utils.Ptr("app not found"),
		})
	}

	accept := ctx.Request().Header.Get(echo.HeaderAccept)
	if accept == MIMEApplicationYAML {
		yaml := composeApp.YAML()
		if yaml == nil {
			return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{
				Message: utils.Ptr("yaml not found"),
			})
		}

		return ctx.String(http.StatusOK, *yaml)
	}

	storeInfo, err := composeApp.StoreInfo()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{
			Message: utils.Ptr(err.Error()),
		})
	}

	apps := map[string]codegen.AppStoreInfo{}

	for _, app := range composeApp.Apps() {
		appStoreInfo, err := app.StoreInfo()
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{
				Message: utils.Ptr(err.Error()),
			})
		}

		apps[app.Name] = *appStoreInfo
	}

	return ctx.JSON(http.StatusOK, codegen.ComposeAppStoreInfoOK{
		Data: &codegen.ComposeAppStoreInfoDetails{
			AppStoreID: storeInfo.AppStoreID,
			MainApp:    storeInfo.MainApp,
			Apps:       &apps,
		},
	})
}
