package v2

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func (a *AppManagement) AppStoreList(ctx echo.Context) error {
	appStoreList := service.MyService.AppStoreManagement().AppStoreList()

	return ctx.JSON(http.StatusOK, codegen.AppStoreListOK{
		Data: &appStoreList,
	})
}

func (a *AppManagement) RegisterAppStore(ctx echo.Context, params codegen.RegisterAppStoreParams) error {
	if params.Url == nil {
		message := "appstore url is required"
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	if _, err := service.MyService.AppStoreManagement().RegisterAppStore(*params.Url); err != nil {
		message := err.Error()
		if err == service.ErrNotAppStore {
			return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
		}

		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.AppStoreRegisterOK{
		Message: utils.Ptr("trying to register app store asynchronously."),
	})
}

func (a *AppManagement) UnregisterAppStore(ctx echo.Context, id codegen.AppStoreID) error {
	appStoreList := service.MyService.AppStoreManagement().AppStoreList()

	if id < 0 || id >= len(appStoreList) {
		message := fmt.Sprintf("app store id %d is not found", id)
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{Message: &message})
	}

	if err := service.MyService.AppStoreManagement().UnregisterAppStore(uint(id)); err != nil {
		message := err.Error()
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.AppStoreUnregisterOK{
		Message: utils.Ptr("app store is unregistered."),
	})
}

func (a *AppManagement) ComposeAppStoreInfoList(ctx echo.Context, params codegen.ComposeAppStoreInfoListParams) error {
	catalog := service.MyService.V2AppStore().Catalog()

	catalog = filterCatalogByCategory(catalog, *params.Category)

	list := lo.MapValues(catalog, func(composeApp *service.ComposeApp, appStoreID string) codegen.ComposeAppStoreInfo {
		storeInfo, err := composeApp.StoreInfo(true)
		if err != nil {
			logger.Error("failed to get store info", zap.Error(err), zap.String("appStoreID", appStoreID))
			return codegen.ComposeAppStoreInfo{}
		}

		return *storeInfo
	})

	return ctx.JSON(http.StatusOK, codegen.ComposeAppStoreInfoListsOK{
		Data: &codegen.ComposeAppStoreInfoLists{
			List: &list,
		},
	})
}

func (a *AppManagement) ComposeAppStoreInfo(ctx echo.Context, id codegen.StoreAppID) error {
	composeApp := service.MyService.V2AppStore().ComposeApp(id)

	if composeApp == nil {
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{
			Message: utils.Ptr("app not found"),
		})
	}

	storeInfo, err := composeApp.StoreInfo(true)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{
			Message: utils.Ptr(err.Error()),
		})
	}

	return ctx.JSON(http.StatusOK, codegen.ComposeAppStoreInfoOK{
		Data: storeInfo,
	})
}

func (a *AppManagement) ComposeApp(ctx echo.Context, id codegen.StoreAppID) error {
	composeApp := service.MyService.V2AppStore().ComposeApp(id)

	if composeApp == nil {
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{
			Message: utils.Ptr("app not found"),
		})
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

	storeInfo, err := composeApp.StoreInfo(false)
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
			Compose:   (*codegen.ComposeApp)(composeApp),
		},
	})
}

func filterCatalogByCategory(catalog map[string]*service.ComposeApp, category string) map[string]*service.ComposeApp {
	if category == "" {
		return catalog
	}

	filteredCatalog := make(map[string]*service.ComposeApp)

	for storeAppID, composeApp := range catalog {
		storeInfo, err := composeApp.StoreInfo(true)
		if err != nil {
			logger.Error("failed to get store info", zap.Error(err), zap.String("storeAppID", storeAppID))
			continue
		}

		mainApp := storeInfo.MainApp
		if mainApp == nil || *mainApp == "" {
			logger.Error("main app is not set", zap.String("storeAppID", storeAppID))
			continue
		}

		if storeInfo.Apps == nil || len(*storeInfo.Apps) == 0 {
			logger.Error("apps is not set", zap.String("storeAppID", storeAppID))
			continue
		}

		mainAppStoreInfo, ok := (*storeInfo.Apps)[*mainApp]
		if !ok {
			logger.Error("main app is not found", zap.String("storeAppID", storeAppID), zap.String("mainApp", *mainApp))
			continue
		}

		if strings.ToLower(mainAppStoreInfo.Category) == strings.ToLower(category) {
			filteredCatalog[storeAppID] = composeApp
		}
	}

	return filteredCatalog
}
