package service

import (
	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/samber/lo"
)

type AppStoreManagement struct {
	onAppStoreRegister   []func(string) error
	onAppStoreUnregister []func(string) error
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

func (a *AppStoreManagement) RegisterAppStore(appstoreURL string) error {
	config.ServerInfo.AppStoreList = append(config.ServerInfo.AppStoreList, appstoreURL)

	if err := config.SaveSetup(); err != nil {
		return err
	}

	for _, fn := range a.onAppStoreRegister {
		if err := fn(appstoreURL); err != nil {
			return err
		}
	}
	return nil
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

func NewAppStoreManagement() *AppStoreManagement {
	appStoreManagement := &AppStoreManagement{}

	return appStoreManagement
}
