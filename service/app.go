package service

import (
	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
)

type App types.ServiceConfig

func (a *App) StoreInfo() (codegen.AppStoreInfo, error) {
	var storeInfo codegen.AppStoreInfo

	ex, ok := a.Extensions[common.ComposeExtensionNameXCasaOS]
	if !ok {
		return storeInfo, ErrComposeExtensionNameXCasaOSNotFound
	}

	if err := loader.Transform(ex, &storeInfo); err != nil {
		return storeInfo, err
	}

	if storeInfo.Scheme == nil || *storeInfo.Scheme == "" {
		storeInfo.Scheme = utils.Ptr(codegen.Http)
	}

	return storeInfo, nil
}
