package v2

import (
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/compose-spec/compose-go/types"
	"github.com/labstack/echo/v4"
)

func (*AppManagement) ComposeAppStoreInfoList(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, codegen.ComposeAppStoreInfoListsOK{
		Data: &codegen.ComposeAppStoreInfoLists{
			// TODO
			Recommend: &[]codegen.ComposeAppStoreInfo{},
			List:      &[]codegen.ComposeAppStoreInfo{},
			Community: &[]codegen.ComposeAppStoreInfo{},
		},
	})
}

func (*AppManagement) ComposeAppStoreInfo(ctx echo.Context, id codegen.AppStoreID) error {
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

	storeInfo.Apps = &apps

	return ctx.JSON(http.StatusOK, codegen.ComposeAppStoreInfoOK{
		Data: storeInfo,
	})
}

func (*AppManagement) ComposeApp(ctx echo.Context, id codegen.AppStoreID) error {
	composeApp := service.MyService.V2AppStore().ComposeApp(id)

	if composeApp == nil {
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{
			Message: utils.Ptr("app not found"),
		})
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

	return ctx.JSON(http.StatusOK, codegen.ComposeAppOK{
		// extension properties aren't marshalled - https://github.com/golang/go/issues/6213
		Data: (*types.Project)(composeApp),
	})
}
