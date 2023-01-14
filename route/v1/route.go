package v1

import "github.com/gin-gonic/gin"

func PropertiesFromQueryParams(httpCtx *gin.Context) map[string]string {
	properties := make(map[string]string)

	for _, param := range httpCtx.Params {
		properties[param.Key] = param.Value
	}

	return properties
}
