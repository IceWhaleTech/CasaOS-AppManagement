package v2

import (
	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
)

type App types.ServiceConfig

func (a *App) StoreInfo() (*codegen.AppStoreInfo, error) {
	if ex, ok := a.Extensions[yamlExtensionName]; ok {
		var storeInfo codegen.AppStoreInfo
		if err := loader.Transform(ex, &storeInfo); err != nil {
			return nil, err
		}
		return &storeInfo, nil
	}
	return nil, ErrYAMLExtensionNotFound
}
