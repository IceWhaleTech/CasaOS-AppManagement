package v2

import (
	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/labstack/echo/v4"
)

type AppManagement struct{}

func NewAppManagement() codegen.ServerInterface {
	return &AppManagement{}
}

func PropertiesFromQueryParams(httpCtx echo.Context) map[string]string {
	properties := make(map[string]string)

	for k, values := range httpCtx.QueryParams() {
		if len(values) > 0 {
			properties[k] = values[0]
		}
	}

	return properties
}

func DefaultQuery(ctx echo.Context, key string, defaultValue string) string {
	if value := ctx.QueryParam(key); value != "" {
		return value
	}

	return defaultValue
}
