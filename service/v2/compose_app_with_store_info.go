package v2

import "github.com/IceWhaleTech/CasaOS-AppManagement/codegen"

type ComposeAppWithStoreInfo struct {
	Compose   *ComposeApp                  `json:"compose,omitempty"`
	StoreInfo *codegen.ComposeAppStoreInfo `json:"store_info,omitempty"`
}
