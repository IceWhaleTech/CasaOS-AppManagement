package v1

import (
	"strings"

	"github.com/docker/docker/api/types"
)

const (
	V1LabelName = "name"
	V1LabelIcon = "icon"
)

func AppName(containerInfo *types.ContainerJSON) string {
	if containerInfo == nil {
		return ""
	}

	if name, ok := containerInfo.Config.Labels[V1LabelName]; ok {
		return name
	}

	return strings.TrimPrefix(containerInfo.Name, "/")
}

func AppIcon(containerInfo *types.ContainerJSON) string {
	if containerInfo == nil {
		return ""
	}

	return containerInfo.Config.Labels[V1LabelIcon]
}
