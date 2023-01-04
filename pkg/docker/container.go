/*
credit: https://github.com/containrrr/watchtower
*/
package docker

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func ImageName(containerInfo *types.ContainerJSON) string {
	imageName := containerInfo.Config.Image

	if !strings.Contains(imageName, ":") {
		imageName = imageName + ":latest"
	}

	return imageName
}

func UpdateContainer(id string, pullAndCheck bool) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx := context.Background()

	containerInfo, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return err
	}

	if pullAndCheck {
		imageName := ImageName(&containerInfo)

		if err := PullNewImage(ctx, imageName); err != nil {
			return err
		}

		currentImageID := containerInfo.ContainerJSONBase.Image

		_, _, err := HasNewImage(ctx, imageName, currentImageID)
		if err != nil {
			return err
		}
	}

	return nil
}
