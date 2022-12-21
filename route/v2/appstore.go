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

	storeInfo, err := composeApp.StoreInfo()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{
			Message: utils.Ptr(err.Error()),
		})
	}

	apps := map[string]codegen.AppStoreInfo{}

	for _, app := range composeApp.Apps() {
		appStoreInfo, err := app.StoreInfo()
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, codegen.BaseResponse{
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
