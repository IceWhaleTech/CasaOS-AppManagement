package v2

import (
	"context"
	"net/http"
	"strings"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	v1 "github.com/IceWhaleTech/CasaOS-AppManagement/service/v1"
	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

func (a *AppManagement) PullImages(ctx echo.Context, params codegen.PullImagesParams) error {
	notificationType := lo.
		If(params.NotificationType != nil, codegen.NotificationType(*params.NotificationType)).
		Else(codegen.NotificationTypeNone)

	backgroundCtx := context.Background()

	if params.ContainerIds != nil {
		for _, containerID := range strings.Split(*params.ContainerIds, ",") {

			containerInfo, err := docker.Container(backgroundCtx, containerID)
			if err != nil {
				logger.Error("get container info failed", zap.Error(err))
				continue
			}

			imageName := docker.ImageName(containerInfo)
			if imageName == "" {
				continue
			}

			appName := v1.AppName(containerInfo)
			appIcon := v1.AppIcon(containerInfo)

			go func() {
				if err := service.MyService.Docker().PullNewImage(imageName, appIcon, appName, notificationType); err != nil {
					logger.Error("pull new image failed", zap.Error(err), zap.String("image", imageName))
				}
			}()
		}
	}

	return ctx.JSON(http.StatusOK, codegen.PullImagesOK{
		Message: utils.Ptr("Images are being pulled asynchronously"),
	})
}
