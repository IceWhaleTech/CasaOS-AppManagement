package service

import (
	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
)

type App types.ServiceConfig

func (a *App) StoreInfo() (*codegen.AppStoreInfo, error) {
	ex, ok := a.Extensions[common.ComposeExtensionNameXCasaOS]
	if !ok {
		return nil, ErrComposeExtensionNameXCasaOSNotFound
	}

	var storeInfo codegen.AppStoreInfo

	if err := loader.Transform(ex, &storeInfo); err != nil {
		return nil, err
	}

	if storeInfo.Container.Scheme == nil || *storeInfo.Container.Scheme == "" {
		storeInfo.Container.Scheme = utils.Ptr(codegen.Http)
	}

	return &storeInfo, nil
}
