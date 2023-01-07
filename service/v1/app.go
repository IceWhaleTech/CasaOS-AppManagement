package v1

import "github.com/docker/docker/api/types"

const (
	V1LabelName = "name"
	V1LabelIcon = "icon"
)

func AppName(containerInfo *types.ContainerJSON) string {
	if containerInfo == nil {
		return ""
	}

	return containerInfo.Config.Labels[V1LabelName]
}

func AppIcon(containerInfo *types.ContainerJSON) string {
	if containerInfo == nil {
		return ""
	}

	return containerInfo.Config.Labels[V1LabelIcon]
}
