package v2

import (
	"fmt"
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"github.com/labstack/echo/v4"
	"gopkg.in/yaml.v3"
)

func (a *AppManagement) Convert(ctx echo.Context, params codegen.ConvertParams) error {
	fileType := codegen.Appfile
	if params.Type != nil {
		fileType = *params.Type
	}

	switch fileType {

	case codegen.Appfile:
		var legacyFile model.CustomizationPostData
		if err := ctx.Bind(&legacyFile); err != nil {
			message := err.Error()
			return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
		}

		compose := legacyFile.Compose()

		yaml, err := yaml.Marshal(compose)
		if err != nil {
			message := err.Error()
			return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
		}

		return ctx.String(http.StatusOK, string(yaml))

	default:
		message := fmt.Sprintf("unsupported file type: %s", fileType)
		return ctx.JSON(http.StatusBadRequest, codegen.ResponseBadRequest{Message: &message})
	}
}
