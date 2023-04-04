package v2

import (
	"fmt"
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/compose-spec/compose-go/types"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

func (a *AppManagement) GetAppGrid(ctx echo.Context) error {
	// v2 Apps
	composeAppsWithStoreInfo, err := composeAppsWithStoreInfo(ctx.Request().Context())
	if err != nil {
		message := err.Error()
		logger.Error("failed to list compose apps with store info", zap.Error(err))
		return ctx.JSON(http.StatusInternalServerError, codegen.ResponseInternalServerError{Message: &message})
	}

	v2AppGridItems := lo.FilterMap(lo.Values(composeAppsWithStoreInfo), func(app codegen.ComposeAppWithStoreInfo, i int) (codegen.WebAppGridItem, bool) {
		item, err := WebAppGridItemAdapterV2(&app)
		if err != nil {
			logger.Error("failed to adapt web app grid item", zap.Error(err), zap.String("app", app.Compose.Name))
			return codegen.WebAppGridItem{}, false
		}

		return *item, true
	})

	// v1 Apps
	casaOSApps, containers := service.MyService.Docker().GetContainerAppList(nil, nil, nil)

	v1AppGridItems := lo.Map(*casaOSApps, func(app model.MyAppList, i int) codegen.WebAppGridItem {
		item, err := WebAppGridItemAdapterV1(&app)
		if err != nil {
			logger.Error("failed to adapt web app grid item", zap.Error(err), zap.String("app", app.Name))
			return codegen.WebAppGridItem{}
		}
		return *item
	})

	// containers
	composeAppContainers := lo.FlatMap(lo.Values(composeAppsWithStoreInfo), func(app codegen.ComposeAppWithStoreInfo, i int) []codegen.ContainerSummary {
		composeApp := (service.ComposeApp)(*app.Compose)
		containers, err := composeApp.Containers(ctx.Request().Context())
		if err != nil {
			logger.Error("failed to get containers for compose app", zap.Error(err), zap.String("app", composeApp.Name))
			return nil
		}
		return lo.Values(containers)
	})

	containerAppGridItems := lo.FilterMap(*containers, func(app model.MyAppList, i int) (codegen.WebAppGridItem, bool) {
		if lo.ContainsBy(composeAppContainers, func(container codegen.ContainerSummary) bool { return container.ID == app.ID }) {
			// already exists as compose app, skipping...
			return codegen.WebAppGridItem{}, false
		}

		item, err := WebAppGridItemAdapterContainer(&app)
		if err != nil {
			logger.Error("failed to adapt web app grid item", zap.Error(err), zap.String("app", app.Name))
			return codegen.WebAppGridItem{}, false
		}
		return *item, true
	})

	// merge v1 and v2 apps
	var appGridItems []codegen.WebAppGridItem
	appGridItems = append(appGridItems, v2AppGridItems...)
	appGridItems = append(appGridItems, v1AppGridItems...)
	appGridItems = append(appGridItems, containerAppGridItems...)

	return ctx.JSON(http.StatusOK, codegen.GetWebAppGridOK{
		Message: utils.Ptr("This data is for internal use ONLY - will not be supported for public use."),
		Data:    &appGridItems,
	})
}

func WebAppGridItemAdapterV2(composeAppWithStoreInfo *codegen.ComposeAppWithStoreInfo) (*codegen.WebAppGridItem, error) {
	if composeAppWithStoreInfo == nil {
		return nil, fmt.Errorf("v2 compose app is nil")
	}

	// validation
	composeApp := (*service.ComposeApp)(composeAppWithStoreInfo.Compose)
	if composeApp == nil {
		return nil, fmt.Errorf("failed to get compose app")
	}

	item := &codegen.WebAppGridItem{
		AppType: codegen.V2app,
		Name:    &composeApp.Name,
		Status:  composeAppWithStoreInfo.Status,
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

	if composeAppStoreInfo.Main == nil || *composeAppStoreInfo.Main == "" {
		return item, fmt.Errorf("failed to get store info for main container app of compose app %s", composeApp.Name)
	}

	var mainApp *types.ServiceConfig
	for i, service := range composeApp.Services {
		if service.Name == *composeAppStoreInfo.Main {
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
	mainAppStoreInfo := (*composeAppStoreInfo.Apps)[*composeAppStoreInfo.Main]
	item.Hostname = mainAppStoreInfo.Hostname
	item.Icon = &composeAppStoreInfo.Icon
	item.Index = &mainAppStoreInfo.Index
	item.Port = &mainAppStoreInfo.PortMap
	item.Scheme = mainAppStoreInfo.Scheme
	item.Title = &composeAppStoreInfo.Title

	// item type
	itemAuthorType := composeApp.AuthorType()
	item.AuthorType = &itemAuthorType

	return item, nil
}

func WebAppGridItemAdapterV1(app *model.MyAppList) (*codegen.WebAppGridItem, error) {
	if app == nil {
		return nil, fmt.Errorf("v1 app is nil")
	}

	item := &codegen.WebAppGridItem{
		AppType:  codegen.V1app,
		Name:     &app.ID,
		Status:   &app.State,
		Image:    &app.Image,
		Hostname: &app.Host,
		Icon:     &app.Icon,
		Index:    &app.Index,
		Port:     &app.Port,
		Scheme:   (*codegen.Scheme)(&app.Protocol),
		Title: &map[string]string{
			"en_US": app.Name,
		},
	}

	return item, nil
}

func WebAppGridItemAdapterContainer(container *model.MyAppList) (*codegen.WebAppGridItem, error) {
	if container == nil {
		return nil, fmt.Errorf("container is nil")
	}

	item := &codegen.WebAppGridItem{
		AppType: codegen.Container,
		Name:    &container.ID,
		Status:  &container.State,
		Image:   &container.Image,
		Title: &map[string]string{
			"en_US": container.Name,
		},
	}

	return item, nil
}
