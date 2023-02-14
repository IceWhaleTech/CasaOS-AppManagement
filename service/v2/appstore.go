package v2

import (
	_ "embed"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
)

type AppStore struct {
	catalog map[string]*ComposeApp
}

//go:embed fixtures/sample.docker-compose.yaml
var SampleComposeAppYAML string

func (s *AppStore) Catalog() map[string]*ComposeApp {
	return s.catalog
}

func (s *AppStore) ComposeApp(appStoreID string) *ComposeApp {
	if composeApp, ok := s.catalog[appStoreID]; ok {
		return composeApp
	}

	return nil
}

func NewAppStore() (*AppStore, error) {
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

	composeAppStoreInfo.AppStoreID = composeAppStoreInfo.MainApp // TODO replace this with real app store ID

	composeApp.Extensions[common.ComposeExtensionNameXCasaOS] = composeAppStoreInfo

	store[*composeAppStoreInfo.AppStoreID] = composeApp

	return store, nil
}
