package v2

import (
	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
)

type App types.ServiceConfig

func (a *App) StoreInfo() (*codegen.AppStoreInfo, error) {
	if ex, ok := a.Extensions[common.ComposeYamlExtensionName]; ok {
		var storeInfo codegen.AppStoreInfo
		if err := loader.Transform(ex, &storeInfo); err != nil {
			return nil, err
		}
		return &storeInfo, nil
	}
	return nil, ErrYAMLExtensionNotFound
}

func (a *App) State() error {
	// TODO implement installation state
	return nil
}
