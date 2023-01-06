package v2

import (
	"net/http"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/labstack/echo/v4"
)

func (a *AppManagement) PullImages(ctx echo.Context, params codegen.PullImagesParams) error {
	images := []string{}

	if params.ContainerIds != nil {
		// TODO get image name from each container id
	}

	if len(images) == 0 {
		return ctx.JSON(http.StatusOK, codegen.PullImagesOK{
			Data: utils.Ptr(false),
		})
	}

	// TODO create pull new image jobs

	panic("implement me")
}
