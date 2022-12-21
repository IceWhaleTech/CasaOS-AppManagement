package v2

import (
	"fmt"

	_ "embed"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
)

const yamlExtensionName = "x-casaos"

type AppStore struct {
	catalog map[string]*ComposeApp
}

var (
	//go:embed fixtures/sample.docker-compose.yaml
	SampleComposeAppYAML string

	ErrYAMLExtensionNotFound = fmt.Errorf("extension `%s` not found", yamlExtensionName)
	ErrMainAppNotFound       = fmt.Errorf("main app not found")
)

func (s *AppStore) Catalog() map[string]*ComposeApp {
	return s.catalog
}

func (s *AppStore) ComposeApp(appStoreID string) *ComposeApp {
	return s.catalog[appStoreID]
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

	project, err := loader.Load(types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{
				Content: []byte(SampleComposeAppYAML),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	project.Extensions["yaml"] = &SampleComposeAppYAML

	if ex, ok := project.Extensions[yamlExtensionName]; ok {
		var storeInfo codegen.ComposeAppStoreInfo
		if err := loader.Transform(ex, &storeInfo); err != nil {
			panic(err)
		}

		store[storeInfo.AppStoreID] = (*ComposeApp)(project)

	} else {
		return nil, ErrYAMLExtensionNotFound
	}

	return store, nil
}
