package v2

import (
	"fmt"
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/compose-spec/compose-go/types"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

func (a *AppManagement) GetAppGrid(ctx echo.Context) error {
	composeAppsWithStoreInfo, err := composeAppsWithStoreInfo(ctx.Request().Context())
	if err != nil {
		message := err.Error()
		logger.Error("failed to list compose apps with store info", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	appGridItems := lo.FilterMap(lo.Values(composeAppsWithStoreInfo), func(app codegen.ComposeAppWithStoreInfo, i int) (codegen.WebAppGridItem, bool) {
		item, err := WebAppGridItemAdapter(app)
		if err != nil {
			logger.Error("failed to adapt web app grid item", zap.Error(err), zap.String("app", app.Compose.Name))
			return codegen.WebAppGridItem{}, false
		}

		return *item, true
	})

	return ctx.JSON(http.StatusOK, codegen.GetWebAppGridOK{
		Message: utils.Ptr("This data is for internal use ONLY - will not be supported for public use."),
		Data:    &appGridItems,
	})
}

func WebAppGridItemAdapter(composeAppWithStoreInfo codegen.ComposeAppWithStoreInfo) (*codegen.WebAppGridItem, error) {
	// validation
	composeApp := (*service.ComposeApp)(composeAppWithStoreInfo.Compose)
	if composeApp == nil {
		return nil, fmt.Errorf("failed to get compose app")
	}

	item := &codegen.WebAppGridItem{
		Name:   &composeApp.Name,
		Status: composeAppWithStoreInfo.Status,
	}

	composeAppStoreInfo := composeAppWithStoreInfo.StoreInfo

	if composeAppStoreInfo == nil {
		return item, fmt.Errorf("failed to get store info for compose app %s", composeApp.Name)
	}

	item.StoreAppID = composeAppStoreInfo.StoreAppID

	// identify the main app
	if composeAppStoreInfo.Apps == nil {
		return item, fmt.Errorf("failed to get container apps for compose app %s", composeApp.Name)
	}

	if composeAppStoreInfo.MainApp == nil || *composeAppStoreInfo.MainApp == "" {
		return item, fmt.Errorf("failed to get store info for main container app of compose app %s", composeApp.Name)
	}

	var mainApp *types.ServiceConfig
	for i, service := range composeApp.Services {
		if service.Name == *composeAppStoreInfo.MainApp {
			mainApp = &composeApp.Services[i]
		}
		break
	}

	// item image
	if mainApp == nil {
		logger.Error("failed to get main app service", zap.String("app", composeApp.Name))
		return item, fmt.Errorf("failed to get main container app for compose app %s", composeApp.Name)
	}
	item.Image = &mainApp.Image

	// item properties from store info
	mainAppStoreInfo := (*composeAppStoreInfo.Apps)[*composeAppStoreInfo.MainApp]
	item.Hostname = mainAppStoreInfo.Container.Hostname
	item.Icon = &mainAppStoreInfo.Icon
	item.Index = &mainAppStoreInfo.Container.Index
	item.Port = &mainAppStoreInfo.Container.PortMap
	item.Scheme = mainAppStoreInfo.Container.Scheme
	item.Title = &mainAppStoreInfo.Title

	// item type
	itemType := composeApp.AuthorType()
	item.Type = &itemType

	return item, nil
}
