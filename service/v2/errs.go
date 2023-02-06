package v2

import (
	"fmt"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
)

var (
	ErrComposeExtensionNameXCasaOSNotFound = fmt.Errorf("extension `%s` not found", common.ComposeExtensionNameXCasaOS)
	ErrComposeExtensionNameYAMLNotFound    = fmt.Errorf("extension `%s` not found", common.ComposeExtensionNameYAML)
	ErrMainAppNotFound                     = fmt.Errorf("main app not found")
	ErrComposeAppNotFound                  = fmt.Errorf("compose app not found")
)
