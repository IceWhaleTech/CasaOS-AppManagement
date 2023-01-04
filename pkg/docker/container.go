/*
credit: https://github.com/containrrr/watchtower
*/
package docker

import (
	"strings"

	"github.com/docker/docker/api/types"
)

func ImageName(containerInfo *types.ContainerJSON) string {
	imageName := containerInfo.Config.Image

	if !strings.Contains(imageName, ":") {
		imageName = imageName + ":latest"
	}

	return imageName
}
