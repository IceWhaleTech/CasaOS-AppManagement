package v2

import (
	"fmt"
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
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

	appGridItems := lo.Map(lo.Values(composeAppsWithStoreInfo), func(app codegen.ComposeAppWithStoreInfo, i int) codegen.WebAppGridItem {
		item, err := webAppGridItemAdapter(app)
		if err != nil {
			logger.Error("failed to adapte web app grid item", zap.Error(err), zap.String("app", app.Compose.Name))
			return codegen.WebAppGridItem{}
		}

		return *item
	})

	return ctx.JSON(http.StatusOK, codegen.GetWebAppGridOK{
		Data: &appGridItems,
	})
}

func webAppGridItemAdapter(app codegen.ComposeAppWithStoreInfo) (*codegen.WebAppGridItem, error) {
	if app.StoreInfo == nil {
		return nil, fmt.Errorf("failed to get store info for compose app %s", app.Compose.Name)
	}

	if app.StoreInfo.Apps == nil {
		return nil, fmt.Errorf("failed to get container apps for compose app %s", app.Compose.Name)
	}

	if app.StoreInfo.MainApp == nil || *app.StoreInfo.MainApp == "" {
		return nil, fmt.Errorf("failed to get store info for main container app of compose app %s", app.Compose.Name)
	}

	mainAppStoreInfo := (*app.StoreInfo.Apps)[*app.StoreInfo.MainApp]

	services := lo.Filter(app.Compose.Services, func(service types.ServiceConfig, i int) bool {
		return service.Name == *app.StoreInfo.MainApp
	})

	if len(services) == 0 {
		logger.Error("failed to get main app service", zap.String("app", app.Compose.Name))
		return nil, fmt.Errorf("failed to get main container app for compose app %s", app.Compose.Name)
	}

	mainApp := services[0]

	itemType := (codegen.WebAppGridItemType)(lo.If(mainAppStoreInfo.Author == common.ComposeAppOfficialAuthor, "official").Else("community"))

	return &codegen.WebAppGridItem{
		Icon:       &mainAppStoreInfo.Icon,
		Image:      &mainApp.Image,
		Hostname:   mainAppStoreInfo.Container.Hostname,
		Index:      &mainAppStoreInfo.Container.Index,
		Port:       &mainAppStoreInfo.Container.PortMap,
		Status:     app.Status,
		StoreAppId: app.StoreInfo.StoreAppID,
		Title:      &mainAppStoreInfo.Title,
		Type:       &itemType,
	}, nil
}
