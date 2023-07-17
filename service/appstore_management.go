package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type AppStoreManagement struct {
	onAppStoreRegister   []func(string) error
	onAppStoreUnregister []func(string) error

	defaultAppStore AppStore
}

func (a *AppStoreManagement) AppStoreList() []codegen.AppStoreMetadata {
	return lo.Map(config.ServerInfo.AppStoreList, func(appStoreURL string, id int) codegen.AppStoreMetadata {
		appStore, err := AppStoreByURL(appStoreURL)
		if err != nil {
			logger.Error("failed to construct appstore", zap.Error(err), zap.String("appstoreURL", appStoreURL))
			return codegen.AppStoreMetadata{}
		}

		workDir, err := appStore.WorkDir()
		if err != nil {
			logger.Error("failed to get appstore workdir", zap.Error(err), zap.String("appstoreURL", appStoreURL))
			return codegen.AppStoreMetadata{}
		}

		storeRoot, err := StoreRoot(workDir)
		if err != nil {
			logger.Error("failed to get appstore storeRoot", zap.Error(err), zap.String("appstoreURL", appStoreURL))
			storeRoot = "internal error - store root not found"
		}

		return codegen.AppStoreMetadata{
			ID:        &id,
			URL:       &appStoreURL,
			StoreRoot: &storeRoot,
		}
	})
}

func (a *AppStoreManagement) OnAppStoreRegister(fn func(string) error) {
	a.onAppStoreRegister = append(a.onAppStoreRegister, fn)
}

func (a *AppStoreManagement) OnAppStoreUnregister(fn func(string) error) {
	a.onAppStoreUnregister = append(a.onAppStoreUnregister, fn)
}

func (a *AppStoreManagement) ChangeGlobal(key string, value string) error {
	config.Global[key] = value

	go func() {
		if err := config.SaveGlobal(); err != nil {
			logger.Error("failed to save global env", zap.Error(err), zap.String("key", key), zap.String("value", value))
			return
		}
	}()

	return nil
}

func (a *AppStoreManagement) DeleteGlobal(key string) error {
	for k := range config.Global {
		if k == key {
			delete(config.Global, k)
		}
	}

	go func() {
		if err := config.SaveGlobal(); err != nil {
			logger.Error("failed to delete global env", zap.Error(err), zap.String("key", key))
			return
		}
	}()

	return nil
}

func (a *AppStoreManagement) RegisterAppStore(ctx context.Context, appstoreURL string, callbacks ...func(*codegen.AppStoreMetadata)) error {
	// check if appstore already exists
	for _, url := range config.ServerInfo.AppStoreList {
		if strings.ToLower(url) == strings.ToLower(appstoreURL) {
			return nil
		}
	}

	appstore, err := AppStoreByURL(appstoreURL)
	if err != nil {
		return err
	}

	go func() {
		go PublishEventWrapper(ctx, common.EventTypeAppStoreRegisterBegin, nil)

		defer PublishEventWrapper(ctx, common.EventTypeAppStoreRegisterEnd, nil)

		var err error

		defer func() {
			if err == nil {
				return
			}

			PublishEventWrapper(ctx, common.EventTypeAppStoreRegisterError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})
		}()

		if err = appstore.UpdateCatalog(); err != nil {
			logger.Error("failed to update appstore catalog", zap.Error(err), zap.String("appstoreURL", appstoreURL))

			return
		}

		// if everything is good, add to the list
		config.ServerInfo.AppStoreList = append(config.ServerInfo.AppStoreList, appstoreURL)

		if err = config.SaveSetup(); err != nil {
			logger.Error("failed to save appstore list", zap.Error(err), zap.String("appstoreURL", appstoreURL))
			return
		}

		for _, fn := range a.onAppStoreRegister {
			if err := fn(appstoreURL); err != nil {
				logger.Error("failed to run onAppStoreRegister", zap.Error(err), zap.String("appstoreURL", appstoreURL))
			}
		}

		appStoreMetadata := &codegen.AppStoreMetadata{
			ID:  utils.Ptr(len(config.ServerInfo.AppStoreList) - 1),
			URL: &appstoreURL,
		}

		for _, callback := range callbacks {
			callback(appStoreMetadata)
		}
	}()

	return nil
}

func (a *AppStoreManagement) UnregisterAppStore(appStoreID uint) error {
	if appStoreID >= uint(len(config.ServerInfo.AppStoreList)) {
		return fmt.Errorf("appstore id %d out of range", appStoreID)
	}

	appStoreURL := config.ServerInfo.AppStoreList[appStoreID]

	// remove appstore from list
	{
		config.ServerInfo.AppStoreList = append(config.ServerInfo.AppStoreList[:appStoreID], config.ServerInfo.AppStoreList[appStoreID+1:]...)

		if err := config.SaveSetup(); err != nil {
			return err
		}
	}

	// remove appstore workdir
	{
		appStore, err := AppStoreByURL(appStoreURL)
		if err != nil {
			return err
		}

		workdir, err := appStore.WorkDir()
		if err != nil {
			logger.Error("error while getting appstore workdir", zap.Error(err), zap.String("url", appStoreURL))
		}

		if len(workdir) != 0 {
			if err := file.RMDir(workdir); err != nil {
				logger.Error("error while removing appstore workdir", zap.Error(err), zap.String("workdir", workdir))
			}
		}
	}

	for _, fn := range a.onAppStoreUnregister {
		if err := fn(appStoreURL); err != nil {
			return err
		}
	}
	return nil
}

func (a *AppStoreManagement) AppStoreMap() (map[string]AppStore, error) {
	appStoreMap := lo.SliceToMap(config.ServerInfo.AppStoreList, func(appStoreURL string) (string, AppStore) {
		appStore, err := AppStoreByURL(appStoreURL)
		if err != nil {
			return "", nil
		}
		return appStoreURL, appStore
	})

	delete(appStoreMap, "")

	return appStoreMap, nil
}

// AppStore interface
func (a *AppStoreManagement) CategoryMap() (map[string]codegen.CategoryInfo, error) {
	appStoreMap, err := a.AppStoreMap()
	if err != nil {
		return nil, err
	}

	allFailed := true

	categoryMap := map[string]codegen.CategoryInfo{}
	for _, appStore := range appStoreMap {
		c, err := appStore.CategoryMap()
		if err != nil {
			logger.Error("error while loading category map", zap.Error(err))
			continue
		}

		allFailed = false

		for name, category := range c {
			categoryMap[name] = category
		}
	}

	if allFailed {
		logger.Info("all appstores failed to load category map, using default")

		categoryMap, err = a.defaultAppStore.CategoryMap()
		if err != nil {
			return nil, err
		}
	}

	for name, category := range categoryMap {
		category.Count = utils.Ptr(0)
		categoryMap[name] = category
	}

	catalog, err := a.Catalog()
	if err != nil {
		return nil, err
	}

	for _, app := range catalog {
		storeInfo, err := app.StoreInfo(false)
		if err != nil {
			continue
		}

		category, ok := categoryMap[storeInfo.Category]
		if !ok {
			continue
		}

		category.Count = lo.ToPtr(*category.Count + 1)

		categoryMap[storeInfo.Category] = category
	}

	return categoryMap, nil
}

func (a *AppStoreManagement) Recommend() ([]string, error) {
	appStoreMap, err := a.AppStoreMap()
	if err != nil {
		logger.Error("error while loading appstore map", zap.Error(err))
		return nil, err
	}

	allFailed := true

	recommend := []string{}
	for _, appStore := range appStoreMap {
		r, err := appStore.Recommend()
		if err != nil {
			logger.Error("error while getting appstore recommend", zap.Error(err))
			continue
		}

		allFailed = false
		recommend = lo.Union(recommend, r)
	}

	if !allFailed {
		return recommend, nil
	}

	logger.Info("No appstore registered")
	if a.defaultAppStore == nil {
		logger.Info("WARNING - no default appstore")
		return nil, nil
	}

	logger.Info("Using default appstore")
	recommend, err = a.defaultAppStore.Recommend()
	if err != nil {
		logger.Error("error while getting default appstore recommend list", zap.Error(err))
		return nil, err
	}

	return recommend, nil
}

func (a *AppStoreManagement) Catalog() (map[string]*ComposeApp, error) {
	catalog := map[string]*ComposeApp{}

	appStoreMap, err := a.AppStoreMap()
	if err != nil {
		return nil, err
	}

	allFailed := true

	for _, appStore := range appStoreMap {

		c, err := appStore.Catalog()
		if err != nil {
			logger.Error("error while getting appstore catalog", zap.Error(err))
			continue
		}

		allFailed = false
		for storeAppID, composeApp := range c {
			catalog[storeAppID] = composeApp
		}
	}

	if !allFailed {
		return catalog, nil
	}

	logger.Info("No appstore registered")
	if a.defaultAppStore == nil {
		logger.Info("WARNING - no default appstore")
		return map[string]*ComposeApp{}, nil
	}

	logger.Info("Using default appstore")
	catalog, err = a.defaultAppStore.Catalog()
	if err != nil {
		return map[string]*ComposeApp{}, err
	}

	return catalog, nil
}

func (a *AppStoreManagement) UpdateCatalog() error {
	appStoreMap, err := a.AppStoreMap()
	if err != nil {
		return err
	}

	for url, appStore := range appStoreMap {
		if err := appStore.UpdateCatalog(); err != nil {
			logger.Error("error while updating catalog for app store", zap.Error(err), zap.String("url", url))
		}
	}

	return nil
}

func (a *AppStoreManagement) ComposeApp(id string) (*ComposeApp, error) {
	appStoreMap, err := a.AppStoreMap()
	if err != nil {
		return nil, err
	}

	for _, appStore := range appStoreMap {
		composeApp, err := appStore.ComposeApp(id)
		if err != nil {
			logger.Error("error while getting appstore compose app", zap.Error(err))
			continue
		}

		if composeApp != nil {
			return composeApp, nil
		}
	}

	logger.Info("app not found in any appstore", zap.String("id", id))

	if a.defaultAppStore == nil {
		logger.Info("WARNING - no default appstore")
		return nil, nil
	}

	logger.Info("Using default appstore")

	composeApp, err := a.defaultAppStore.ComposeApp(id)
	if err != nil {
		return nil, err
	}

	return composeApp, nil
}

func (a *AppStoreManagement) WorkDir() (string, error) {
	panic("not implemented and will never be implemented - this is a virtual appstore")
}

func NewAppStoreManagement() *AppStoreManagement {
	defaultAppStore, err := NewDefaultAppStore()
	if err != nil {
		fmt.Printf("error while loading default appstore: %s\n", err.Error())
	}

	appStoreManagement := &AppStoreManagement{
		defaultAppStore: defaultAppStore,
	}

	return appStoreManagement
}
