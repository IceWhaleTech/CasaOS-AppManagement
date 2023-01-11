package v2

import (
	"fmt"

	_ "embed"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
)

type AppStore struct {
	catalog map[string]*ComposeApp
}

var (
	//go:embed fixtures/sample.docker-compose.yaml
	SampleComposeAppYAML string

	ErrComposeExtensionNameXCasaOSNotFound = fmt.Errorf("extension `%s` not found", common.ComposeExtensionNameXCasaOS)
	ErrMainAppNotFound                     = fmt.Errorf("main app not found")
)

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

	composeApp, err := NewComposeAppFromYAML([]byte(SampleComposeAppYAML))
	if err != nil {
		return nil, err
	}

	composeAppStoreInfo, err := composeApp.StoreInfo()
	if err != nil {
		return nil, err
	}

	store[*composeAppStoreInfo.AppStoreID] = composeApp

	return store, nil
}
