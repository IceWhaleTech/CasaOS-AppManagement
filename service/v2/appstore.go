package v2

import (
	"fmt"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"

	_ "embed"
)

const extensionName = "x-casaos"

var (
	//go:embed fixtures/sample.docker-compose.yaml
	SampleComposeAppYAML string

	Store = map[string]*ComposeApp{}

	ErrExtensionNotFound = fmt.Errorf("extension `%s` not found", extensionName)
)

type (
	App   types.ServiceConfig
	AppEx struct {
		Title map[string]string `mapstructure:"title"`
		Name  string            `mapstructure:"name"`
	}
)

func (a *App) StoreInfo() (*AppEx, error) {
	if ex, ok := a.Extensions[extensionName]; ok {
		var appEx AppEx
		if err := loader.Transform(ex, &appEx); err != nil {
			return nil, err
		}
		return &appEx, nil
	}
	return nil, ErrExtensionNotFound
}

type (
	ComposeApp   types.Project
	ComposeAppEx struct {
		StoreAppID string `mapstructure:"store_appid"`
	}
)

func (a *ComposeApp) StoreInfo() (*ComposeAppEx, error) {
	if ex, ok := a.Extensions["x-casaos"]; ok {
		var appEx ComposeAppEx
		if err := loader.Transform(ex, &appEx); err != nil {
			return nil, err
		}
		return &appEx, nil
	}
	return nil, ErrExtensionNotFound
}

func (a *ComposeApp) YAML() *string {
	if yaml, ok := a.Extensions["yaml"]; ok {
		return yaml.(*string)
	}
	return nil
}

func init() {
	project, err := loader.Load(types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{
				Content: []byte(SampleComposeAppYAML),
			},
		},
	})
	if err != nil {
		panic(err)
	}

	project.Extensions["yaml"] = &SampleComposeAppYAML

	if ex, ok := project.Extensions["x-casaos"]; ok {
		var projectEx ComposeAppEx
		if err := loader.Transform(ex, &projectEx); err != nil {
			panic(err)
		}

		Store[projectEx.StoreAppID] = (*ComposeApp)(project)

	} else {
		panic("invalid project extension")
	}
}

func GetComposeApp(storeAppID string) *ComposeApp {
	return Store[storeAppID]
}

func GetApp(storeAppID, name string) *App {
	if composeApp, ok := Store[storeAppID]; ok {
		for i, service := range composeApp.Services {
			if service.Name == name {
				return (*App)(&composeApp.Services[i])
			}
		}
	}
	return nil
}
