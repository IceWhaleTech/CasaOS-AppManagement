package v1

import (
	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
)

func YAML(ctx echo.Context, code int, i interface{}) error {
	ctx.Response().WriteHeader(code)
	ctx.Response().Header().Set(echo.HeaderContentType, "text/yaml")

	return yaml.NewEncoder(ctx.Response()).Encode(i)
}