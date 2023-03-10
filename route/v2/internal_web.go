package v2

import (
	"fmt"
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
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
		item, err := WebAppGridItemAdapter(app)
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

func WebAppGridItemAdapter(composeAppWithStoreInfo codegen.ComposeAppWithStoreInfo) (*codegen.WebAppGridItem, error) {
	// validation
	composeApp := (*service.ComposeApp)(composeAppWithStoreInfo.Compose)
	if composeApp == nil {
		return nil, fmt.Errorf("failed to get compose app")
	}

	composeAppStoreInfo := composeAppWithStoreInfo.StoreInfo

	if composeAppStoreInfo == nil {
		return nil, fmt.Errorf("failed to get store info for compose app %s", composeApp.Name)
	}

	if composeAppStoreInfo.Apps == nil {
		return nil, fmt.Errorf("failed to get container apps for compose app %s", composeApp.Name)
	}

	if composeAppStoreInfo.MainApp == nil || *composeAppStoreInfo.MainApp == "" {
		return nil, fmt.Errorf("failed to get store info for main container app of compose app %s", composeApp.Name)
	}

	// identify the main app
	var mainApp *types.ServiceConfig
	for i, service := range composeApp.Services {
		if service.Name == *composeAppStoreInfo.MainApp {
			mainApp = &composeApp.Services[i]
		}
		break
	}

	if mainApp == nil {
		logger.Error("failed to get main app service", zap.String("app", composeApp.Name))
		return nil, fmt.Errorf("failed to get main container app for compose app %s", composeApp.Name)
	}

	// item type
	mainAppStoreInfo := (*composeAppStoreInfo.Apps)[*composeAppStoreInfo.MainApp]
	itemType := (codegen.WebAppGridItemType)(lo.If(mainAppStoreInfo.Author == common.ComposeAppOfficialAuthor, "official").Else("community"))

	return &codegen.WebAppGridItem{
		Name:       &composeApp.Name,
		Status:     composeAppWithStoreInfo.Status,
		StoreAppId: composeAppStoreInfo.StoreAppID,
		Type:       &itemType,

		Hostname: mainAppStoreInfo.Container.Hostname,
		Icon:     &mainAppStoreInfo.Icon,
		Image:    &mainApp.Image,
		Index:    &mainAppStoreInfo.Container.Index,
		Port:     &mainAppStoreInfo.Container.PortMap,
		Scheme:   mainAppStoreInfo.Container.Scheme,
		Title:    &mainAppStoreInfo.Title,
	}, nil
}
