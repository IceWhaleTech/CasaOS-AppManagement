/*
credit: https://github.com/containrrr/watchtower
*/
package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/IceWhaleTech/CasaOS-Common/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/samber/lo"
)

func ImageName(containerInfo *types.ContainerJSON) string {
	imageName := containerInfo.Config.Image

	if !strings.Contains(imageName, ":") {
		imageName = imageName + ":latest"
	}

	return imageName
}

func Container(ctx context.Context, id string) (*types.ContainerJSON, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	containerInfo, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}

	return &containerInfo, nil
}

func CloneContainer(ctx context.Context, id string, newName string) (string, error) {
	containerInfo, err := Container(ctx, id)
	if err != nil {
		return "", err
	}

	imageInfo, err := Image(ctx, containerInfo.Image)
	if err != nil {
		return "", err
	}

	config := runtimeConfig(containerInfo, imageInfo)
	hostConfig := hostConfig(containerInfo)
	networkConfig := &network.NetworkingConfig{EndpointsConfig: containerInfo.NetworkSettings.Networks}
	simpleNetworkConfig := simpleNetworkConfig(networkConfig)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}
	defer cli.Close()

	newContainer, err := cli.ContainerCreate(ctx, config, hostConfig, simpleNetworkConfig, nil, newName)
	if err != nil {
		return "", err
	}

	if !(hostConfig.NetworkMode.IsHost()) {
		for k := range simpleNetworkConfig.EndpointsConfig {
			if err := cli.NetworkDisconnect(ctx, k, newContainer.ID, true); err != nil {
				return newContainer.ID, err
			}
		}

		for k, v := range networkConfig.EndpointsConfig {
			if err := cli.NetworkConnect(ctx, k, newContainer.ID, v); err != nil {
				return newContainer.ID, err
			}
		}
	}

	return newContainer.ID, nil
}

func RemoveContainer(ctx context.Context, id string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	return cli.ContainerRemove(ctx, id, types.ContainerRemoveOptions{Force: true})
}

func RenameContainer(ctx context.Context, id string, name string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	return cli.ContainerRename(ctx, id, name)
}

func StartContainer(ctx context.Context, id string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	containerInfo, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return err
	}

	if !containerInfo.State.Running {
		return cli.ContainerStart(ctx, id, types.ContainerStartOptions{})
	}

	return nil
}

func StopContainer(ctx context.Context, id string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	containerInfo, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return err
	}

	if containerInfo.State.Running {
		if err := cli.ContainerStop(ctx, id, container.StopOptions{}); err != nil {
			return err
		}

		if err := WaitContainer(ctx, id, container.WaitConditionNotRunning); err != nil {
			return err
		}
	}

	return nil
}

func WaitContainer(ctx context.Context, id string, condition container.WaitCondition) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	wait, errChan := cli.ContainerWait(ctx, id, condition)
	for {
		select {
		case err := <-errChan:
			return err
		case <-wait:
			return nil
		}
	}
}

func runtimeConfig(containerInfo *types.ContainerJSON, imageInfo *types.ImageInspect) *container.Config {
	config := containerInfo.Config
	hostConfig := containerInfo.HostConfig
	imageConfig := imageInfo.Config

	if config.WorkingDir == imageConfig.WorkingDir {
		config.WorkingDir = ""
	}

	if config.User == imageConfig.User {
		config.User = ""
	}

	if hostConfig.NetworkMode.IsContainer() {
		config.Hostname = ""
	}

	if utils.CompareStringSlices(config.Entrypoint, imageConfig.Entrypoint) {
		config.Entrypoint = nil
		if utils.CompareStringSlices(config.Cmd, imageConfig.Cmd) {
			config.Cmd = nil
		}
	}

	config.Env = lo.Filter(config.Env, func(s string, i int) bool { return !lo.Contains(imageConfig.Env, s) })

	config.Labels = lo.OmitBy(config.Labels, func(k string, v string) bool {
		v2, ok := imageConfig.Labels[k]
		return ok && v == v2
	})

	config.Volumes = lo.OmitBy(config.Volumes, func(k string, v struct{}) bool {
		v2, ok := imageConfig.Volumes[k]
		return ok && v == v2
	})

	// subtract ports exposed in image from container
	for k := range config.ExposedPorts {
		if _, ok := imageConfig.ExposedPorts[k]; ok {
			delete(config.ExposedPorts, k)
		}
	}

	for p := range containerInfo.HostConfig.PortBindings {
		config.ExposedPorts[p] = struct{}{}
	}

	config.Image = ImageName(containerInfo)
	return config
}

func hostConfig(containerInfo *types.ContainerJSON) *container.HostConfig {
	hostConfig := containerInfo.HostConfig

	for i, link := range hostConfig.Links {
		name := link[0:strings.Index(link, ":")]
		alias := link[strings.LastIndex(link, "/"):]

		hostConfig.Links[i] = fmt.Sprintf("%s:%s", name, alias)
	}

	return hostConfig
}

// simpleNetworkConfig is a networkConfig with only 1 network.
// see: https://github.com/docker/docker/issues/29265
func simpleNetworkConfig(networkConfig *network.NetworkingConfig) *network.NetworkingConfig {
	oneEndpoint := make(map[string]*network.EndpointSettings)
	for k, v := range networkConfig.EndpointsConfig {
		oneEndpoint[k] = v
		// we only need 1
		break
	}
	return &network.NetworkingConfig{EndpointsConfig: oneEndpoint}
}
