package service

import (
	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/samber/lo"
)

type AppStoreManagement struct {
	onAppStoreRegister   []func([]string) error
	onAppStoreUnregister []func(string) error
}

func (s *AppStoreManagement) AppStoreList() []codegen.AppStoreMetadata {
	return lo.Map(config.ServerInfo.AppStoreList, func(appStoreURL string, id int) codegen.AppStoreMetadata {
		return codegen.AppStoreMetadata{
			ID:  &id,
			URL: &appStoreURL,
		}
	})
}

func (s *AppStoreManagement) OnAppStoreRegister(fn func([]string) error) {
	s.onAppStoreRegister = append(s.onAppStoreRegister, fn)
}

func (s *AppStoreManagement) OnAppStoreUnregister(fn func(string) error) {
	s.onAppStoreUnregister = append(s.onAppStoreUnregister, fn)
}

func (s *AppStoreManagement) RegisterAppStore(appstoreURL string) error {
	config.ServerInfo.AppStoreList = append(config.ServerInfo.AppStoreList, appstoreURL)

	for _, fn := range s.onAppStoreRegister {
		if err := fn(config.ServerInfo.AppStoreList); err != nil {
			return err
		}
	}
	return nil
}

func (s *AppStoreManagement) UnregisterAppStore(appStoreID uint) error {
	appStoreURL := config.ServerInfo.AppStoreList[appStoreID]

	config.ServerInfo.AppStoreList = append(config.ServerInfo.AppStoreList[:appStoreID], config.ServerInfo.AppStoreList[appStoreID+1:]...)

	for _, fn := range s.onAppStoreUnregister {
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
