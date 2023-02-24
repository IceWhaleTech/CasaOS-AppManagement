package service

import (
	_ "embed"
	"net/url"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
)

type AppStore struct {
	url     string
	catalog map[string]*ComposeApp
}

//go:embed fixtures/sample.docker-compose.yaml
var SampleComposeAppYAML string

func (s *AppStore) UpdateCatalog() error {
	if _, err := url.Parse(s.url); err != nil {
		return err
	}

	// TODO - implement this

	return nil
}

func (s *AppStore) Catalog() map[string]*ComposeApp {
	return s.catalog
}

func (s *AppStore) ComposeApp(appStoreID string) *ComposeApp {
	if composeApp, ok := s.catalog[appStoreID]; ok {
		return composeApp
	}

	return nil
}

func NewAppStore(url string) *AppStore {
	return &AppStore{
		url:     url,
		catalog: map[string]*ComposeApp{},
	}
}

func NewAppStoreForTest() (*AppStore, error) {
	store, err := tempStoreForTest() // TODO - replace this with real store
	if err != nil {
		return nil, err
	}

	return &AppStore{
		catalog: store,
	}, nil
}

func tempStoreForTest() (map[string]*ComposeApp, error) {
	store := map[string]*ComposeApp{}

	composeApp, err := NewComposeAppFromYAML([]byte(SampleComposeAppYAML), nil)
	if err != nil {
		return nil, err
	}

	composeAppStoreInfo, err := composeApp.StoreInfo(false)
	if err != nil {
		return nil, err
	}

	composeAppStoreInfo.StoreAppID = composeAppStoreInfo.MainApp // TODO replace this with real app store ID

	composeApp.Extensions[common.ComposeExtensionNameXCasaOS] = composeAppStoreInfo

	store[*composeAppStoreInfo.StoreAppID] = composeApp

	return store, nil
}
