package service

import (
	"fmt"
	"strings"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
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

	defaultAppStore *AppStore
}

func (a *AppStoreManagement) AppStoreList() []codegen.AppStoreMetadata {
	return lo.Map(config.ServerInfo.AppStoreList, func(appStoreURL string, id int) codegen.AppStoreMetadata {
		appStore, err := NewAppStore(appStoreURL)
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

func (a *AppStoreManagement) RegisterAppStore(appstoreURL string) (chan *codegen.AppStoreMetadata, error) {
	appstoreURL = strings.ToLower(appstoreURL)

	// check if appstore already exists
	config.ServerInfo.AppStoreList = lo.Map(config.ServerInfo.AppStoreList,
		func(url string, id int) string {
			return strings.ToLower(url)
		})

	for _, url := range config.ServerInfo.AppStoreList {
		if url == appstoreURL {
			return nil, nil
		}
	}

	appstore, err := NewAppStore(appstoreURL)
	if err != nil {
		return nil, err
	}

	c := make(chan *codegen.AppStoreMetadata)

	go func() {
		if err := appstore.UpdateCatalog(); err != nil {
			logger.Error("failed to update appstore catalog", zap.Error(err), zap.String("appstoreURL", appstoreURL))
			return
		}

		// if everything is good, add to the list
		config.ServerInfo.AppStoreList = append(config.ServerInfo.AppStoreList, appstoreURL)

		if err := config.SaveSetup(); err != nil {
			logger.Error("failed to save appstore list", zap.Error(err), zap.String("appstoreURL", appstoreURL))
			return
		}

		for _, fn := range a.onAppStoreRegister {
			if err := fn(appstoreURL); err != nil {
				logger.Error("failed to run onAppStoreRegister", zap.Error(err), zap.String("appstoreURL", appstoreURL))
			}
		}

		c <- &codegen.AppStoreMetadata{
			ID:  utils.Ptr(len(config.ServerInfo.AppStoreList) - 1),
			URL: &appstoreURL,
		}
	}()

	return c, nil
}

func (a *AppStoreManagement) UnregisterAppStore(appStoreID uint) error {
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
		appStore, err := NewAppStore(appStoreURL)
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

func (a *AppStoreManagement) Recommend() []string {
	appStoreMap, err := a.AppStoreMap()
	if err != nil {
		logger.Error("error while loading appstore map", zap.Error(err))
		return []string{}
	}

	recommend := []string{}
	for _, appStore := range appStoreMap {
		recommend = lo.Union(recommend, appStore.Recommend())
	}

	return recommend
}

func (a *AppStoreManagement) Catalog() map[string]*ComposeApp {
	catalog := map[string]*ComposeApp{}

	appStoreMap, err := a.AppStoreMap()
	if err != nil {
		logger.Error("error while loading appstore map", zap.Error(err))
		return map[string]*ComposeApp{}
	}

	for _, appStore := range appStoreMap {
		for storeAppID, composeApp := range appStore.Catalog() {
			catalog[storeAppID] = composeApp
		}
	}

	if len(catalog) == 0 {
		logger.Info("No appstore registered")
		if a.defaultAppStore == nil {
			logger.Info("WARNING - no default appstore")
			return map[string]*ComposeApp{}
		}

		logger.Info("Using default appstore")
		return a.defaultAppStore.Catalog()
	}

	return catalog
}

func (a *AppStoreManagement) UpdateCatalog() {
	appStoreMap, err := a.AppStoreMap()
	if err != nil {
		logger.Error("error while loading appstore map", zap.Error(err))
		return
	}

	for url, appStore := range appStoreMap {
		if err := appStore.UpdateCatalog(); err != nil {
			logger.Error("error while updating catalog for app store", zap.Error(err), zap.String("url", url))
		}
	}
}

func (a *AppStoreManagement) ComposeApp(id string) *ComposeApp {
	appStoreMap, err := a.AppStoreMap()
	if err != nil {
		logger.Error("error while loading appstore map", zap.Error(err))
		return nil
	}

	for _, appStore := range appStoreMap {
		if composeApp := appStore.ComposeApp(id); composeApp != nil {
			return composeApp
		}
	}

	logger.Info("No appstore registered")

	if a.defaultAppStore == nil {
		logger.Info("WARNING - no default appstore")
		return nil
	}

	logger.Info("Using default appstore")

	return a.defaultAppStore.ComposeApp(id)
}

func (a *AppStoreManagement) AppStoreMap() (map[string]*AppStore, error) {
	appStoreMap := lo.SliceToMap(config.ServerInfo.AppStoreList, func(appStoreURL string) (string, *AppStore) {
		appStore, err := NewAppStore(appStoreURL)
		if err != nil {
			return "", nil
		}
		return appStoreURL, appStore
	})

	return appStoreMap, nil
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
