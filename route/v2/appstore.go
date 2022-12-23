package v2

import (
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/compose-spec/compose-go/types"
	"github.com/labstack/echo/v4"
)

func (*AppManagement) AppList(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, codegen.ComposeAppStoreInfoListsOK{
		Data: &codegen.ComposeAppStoreInfoLists{
			// TODO
			Recommend: &[]codegen.ComposeAppStoreInfo{},
			List:      &[]codegen.ComposeAppStoreInfo{},
			Community: &[]codegen.ComposeAppStoreInfo{},
		},
	})
}

func (*AppManagement) AppInfo(ctx echo.Context, id codegen.AppStoreID) error {
	composeApp := service.MyService.V2AppStore().ComposeApp(id)

	if composeApp == nil {
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{
			Message: utils.Ptr("app not found"),
		})
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
		Data: &codegen.ComposeAppStoreInfo{
			AppStoreID: storeInfo.AppStoreID,
			MainApp:    storeInfo.MainApp,
			Apps:       &apps,
		},
	})
}

func (*AppManagement) AppCompose(ctx echo.Context, id codegen.AppStoreID) error {
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

	return ctx.JSON(http.StatusOK, codegen.ComposeAppOK{
		Data: (*types.Project)(composeApp),
	})
}