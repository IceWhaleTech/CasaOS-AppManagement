package v2

import (
	"fmt"
	"regexp"
	"strings"

	_ "embed"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
)

type AppStore struct {
	catalog map[string]*ComposeApp
}

var (
	//go:embed fixtures/sample.docker-compose.yaml
	SampleComposeAppYAML string

	ErrYAMLExtensionNotFound = fmt.Errorf("extension `%s` not found", common.ComposeYamlExtensionName)
	ErrMainAppNotFound       = fmt.Errorf("main app not found")
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

func LoadComposeApp(yaml []byte) (*ComposeApp, error) {
	project, err := loader.Load(
		types.ConfigDetails{
			ConfigFiles: []types.ConfigFile{
				{
					Content: []byte(yaml),
				},
			},
			Environment: map[string]string{},
		},
		func(o *loader.Options) { o.SkipInterpolation = true },
	)
	if err != nil {
		return nil, err
	}

	project.Extensions["yaml"] = &SampleComposeAppYAML

	return (*ComposeApp)(project), nil
}

func tempStoreForTest() (map[string]*ComposeApp, error) {
	store := map[string]*ComposeApp{}

	composeApp, err := LoadComposeApp([]byte(SampleComposeAppYAML))
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

var nonAlphaNumeric = regexp.MustCompile(`[^a-z0-9]+`)

func Standardize(text string) string {
	result := strings.ToLower(text)

	// Replace any non-alphanumeric characters with a single hyphen
	result = nonAlphaNumeric.ReplaceAllString(result, "-")

	for strings.Contains(result, "--") {
		result = strings.Replace(result, "--", "-", -1)
	}

	// Remove any leading or trailing hyphens
	result = strings.Trim(result, "-")

	return result
}
