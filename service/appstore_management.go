package service

import (
	"strings"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/samber/lo"
)

type AppStoreManagement struct {
	onAppStoreRegister   []func(string) error
	onAppStoreUnregister []func(string) error

	appStoreMap map[string]*AppStore
}

func (a *AppStoreManagement) AppStoreList() []codegen.AppStoreMetadata {
	return lo.Map(config.ServerInfo.AppStoreList, func(appStoreURL string, id int) codegen.AppStoreMetadata {
		return codegen.AppStoreMetadata{
			ID:  &id,
			URL: &appStoreURL,
		}
	})
}

func (a *AppStoreManagement) OnAppStoreRegister(fn func(string) error) {
	a.onAppStoreRegister = append(a.onAppStoreRegister, fn)
}

func (a *AppStoreManagement) OnAppStoreUnregister(fn func(string) error) {
	a.onAppStoreUnregister = append(a.onAppStoreUnregister, fn)
}

func (a *AppStoreManagement) RegisterAppStore(appstoreURL string) (*codegen.AppStoreMetadata, error) {
	appstoreURL = strings.ToLower(appstoreURL)

	// check if appstore already exists
	config.ServerInfo.AppStoreList = lo.Map(config.ServerInfo.AppStoreList,
		func(url string, id int) string {
			return strings.ToLower(url)
		})

	for i, url := range config.ServerInfo.AppStoreList {
		if url == appstoreURL {
			return &codegen.AppStoreMetadata{
				ID:  &i,
				URL: &config.ServerInfo.AppStoreList[i],
			}, nil
		}
	}

	// try to clone the store locally
	appstore := NewAppStore(appstoreURL)
	if err := appstore.UpdateCatalog(); err != nil {
		// TODO clean up

		return nil, err
	}

	// if everything is good, add to the list
	config.ServerInfo.AppStoreList = append(config.ServerInfo.AppStoreList, appstoreURL)

	if err := config.SaveSetup(); err != nil {
		return nil, err
	}

	for _, fn := range a.onAppStoreRegister {
		if err := fn(appstoreURL); err != nil {
			return nil, err
		}
	}

	return &codegen.AppStoreMetadata{
		ID:  utils.Ptr(len(config.ServerInfo.AppStoreList) - 1),
		URL: &appstoreURL,
	}, nil
}

func (a *AppStoreManagement) UnregisterAppStore(appStoreID uint) error {
	appStoreURL := config.ServerInfo.AppStoreList[appStoreID]

	config.ServerInfo.AppStoreList = append(config.ServerInfo.AppStoreList[:appStoreID], config.ServerInfo.AppStoreList[appStoreID+1:]...)

	if err := config.SaveSetup(); err != nil {
		return err
	}

	for _, fn := range a.onAppStoreUnregister {
		if err := fn(appStoreURL); err != nil {
			return err
		}
	}
	return nil
}

func (a *AppStoreManagement) addAppStore(url string) error {
	appStore := NewAppStore(url)

	a.appStoreMap[url] = appStore

	return nil
}

func NewAppStoreManagement() *AppStoreManagement {
	appStoreManagement := &AppStoreManagement{
		appStoreMap: map[string]*AppStore{},
	}

	appStoreManagement.OnAppStoreRegister(appStoreManagement.addAppStore)

	return appStoreManagement
}
