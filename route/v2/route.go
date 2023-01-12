package v2

import (
	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/labstack/echo/v4"
)

type AppManagement struct{}

func NewAppManagement() codegen.ServerInterface {
	return &AppManagement{}
}

// There are 3 types of contexts here:
//
// - context.Context
//
// - echo.Context
//
// - context key/value pairs for our own use
//
// This func extract context for our own use from echo.Context, and attach it to context.Context, so it can be passed on...
func ContextMapFromQueryParams(httpCtx echo.Context) map[string]string {
	contextMap := make(map[string]string)

	for k, values := range httpCtx.QueryParams() {
		if len(values) > 0 {
			contextMap[k] = values[0]
		}
	}

	return contextMap
}
