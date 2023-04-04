package service

import (
	"fmt"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
)

var (
	ErrComposeAppNotFound                  = fmt.Errorf("compose app not found")
	ErrComposeAppNotMatch                  = fmt.Errorf("compose app not match")
	ErrComposeExtensionNameXCasaOSNotFound = fmt.Errorf("extension `%s` not found", common.ComposeExtensionNameXCasaOS)
	ErrComposeFileNotFound                 = fmt.Errorf("compose file not found")
	ErrInvalidComposeAppStatus             = fmt.Errorf("invalid compose app status")
	ErrMainAppNotFound                     = fmt.Errorf("main app not found")
	ErrNotFoundInAppStore                  = fmt.Errorf("not found in app store")
	ErrSetStoreAppID                       = fmt.Errorf("failed to set store app ID")
	ErrStoreInfoNotFound                   = fmt.Errorf("store info not found")
)
