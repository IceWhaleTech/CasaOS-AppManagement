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
	if params.Url == nil || *params.Url == "" {
		message := "appstore url is required"
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}

	isExist := lo.ContainsBy(service.MyService.AppStoreManagement().AppStoreList(), func(appstore codegen.AppStoreMetadata) bool {
		return appstore.URL != nil && strings.ToLower(*appstore.URL) == strings.ToLower(*params.Url)
	})

	if isExist {
		message := "appstore is already registered"
		return ctx.JSON(http.StatusOK, codegen.AppStoreRegisterOK{Message: &message})
	}

	if _, err := service.MyService.AppStoreManagement().RegisterAppStore(*params.Url); err != nil {
		message := err.Error()
		if err == service.ErrNotAppStore {
			return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
		}

		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	return ctx.JSON(http.StatusOK, codegen.AppStoreRegisterOK{
		Message: utils.Ptr("trying to register app store asynchronously - might fail if app store cannot be validated."),
	})
}

func (a *AppManagement) UnregisterAppStore(ctx echo.Context, id codegen.AppStoreID) error {
	appStoreList := service.MyService.AppStoreManagement().AppStoreList()

	if id < 0 || id >= len(appStoreList) {
		message := fmt.Sprintf("app store id %d is not found", id)
		return ctx.JSON(http.StatusNotFound, codegen.ResponseNotFound{Message: &message})
	}

	if len(appStoreList) == 1 {
		message := "cannot unregister the last app store - need at least one app store"
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
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

	if params.Category != nil {
		catalog = FilterCatalogByCategory(catalog, *params.Category)
	}

	// list
	list := lo.MapValues(catalog, func(composeApp *service.ComposeApp, appStoreID string) codegen.ComposeAppStoreInfo {
		storeInfo, err := composeApp.StoreInfo(true)
		if err != nil {
			logger.Error("failed to get store info", zap.Error(err), zap.String("appStoreID", appStoreID))
			return codegen.ComposeAppStoreInfo{}
		}

		return *storeInfo
	})

	data := &codegen.ComposeAppStoreInfoLists{
		List: &list,
	}

	// recommend
	recommend := service.MyService.V2AppStore().Recommend()
	if params.Category != nil {
		recommend = lo.Intersect(recommend, lo.Keys(catalog))
	}

	data.Recommend = &recommend

	// installed
	installedComposeApps, err := service.MyService.Compose().List(ctx.Request().Context())
	if err != nil {
		message := err.Error()
		logger.Error("failed to list installed compose apps", zap.Error(err))
		return ctx.JSON(http.StatusOK, codegen.ComposeAppStoreInfoListsOK{
			Message: &message,
			Data:    data,
		})
	}

	installed := lo.FilterMap(lo.Values(installedComposeApps), func(composeApp *service.ComposeApp, i int) (string, bool) {
		storeInfo, err := composeApp.StoreInfo(false)
		if err != nil {
			logger.Error("failed to get store info", zap.Error(err), zap.String("name", composeApp.Name))
			return "", false
		}

		if storeInfo == nil {
			logger.Error("failed to get store info - nil value", zap.String("name", composeApp.Name))
			return "", false
		}
		return *storeInfo.StoreAppID, true
	})

	data.Installed = &installed

	return ctx.JSON(http.StatusOK, codegen.ComposeAppStoreInfoListsOK{Data: data})
}

func (a *AppManagement) ComposeAppStoreInfo(ctx echo.Context, id codegen.StoreAppIDString) error {
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

func (a *AppManagement) ComposeApp(ctx echo.Context, id codegen.StoreAppIDString) error {
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

func FilterCatalogByCategory(catalog map[string]*service.ComposeApp, category string) map[string]*service.ComposeApp {
	if category == "" {
		return catalog
	}

	return lo.PickBy(catalog, func(storeAppID string, composeApp *service.ComposeApp) bool {
		storeInfo, err := composeApp.StoreInfo(true)
		if err != nil {
			return false
		}

		mainApp := storeInfo.MainApp
		if mainApp == nil || *mainApp == "" {
			return false
		}

		if storeInfo.Apps == nil || len(*storeInfo.Apps) == 0 {
			return false
		}

		mainAppStoreInfo, ok := (*storeInfo.Apps)[*mainApp]
		if !ok {
			return false
		}

		return strings.ToLower(mainAppStoreInfo.Category) == strings.ToLower(category)
	})
}
