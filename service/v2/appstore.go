package v2

import (
	_ "embed"
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

	composeApp, err := NewComposeAppFromYAML([]byte(SampleComposeAppYAML))
	if err != nil {
		return nil, err
	}

	composeAppStoreInfo, err := composeApp.StoreInfo()
	if err != nil {
		return nil, err
	}

	composeAppStoreInfo.AppStoreID = composeAppStoreInfo.MainApp // TODO remove this line

	store[*composeAppStoreInfo.AppStoreID] = composeApp

	return store, nil
}
