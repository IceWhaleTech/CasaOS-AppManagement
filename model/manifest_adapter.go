package model

import (
	"strconv"
	"strings"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/compose-spec/compose-go/types"
)

func (p *PortArray) ToServicePorts() []types.ServicePortConfig {
	ports := []types.ServicePortConfig{}

	for _, port := range *p {
		target, err := strconv.Atoi(port.ContainerPort)
		if err != nil {
			continue
		}

		ports = append(ports, types.ServicePortConfig{
			Target:    uint32(target),
			Published: port.CommendPort,
			Protocol:  port.Protocol,
		})
	}
	return ports
}

func (p *PathArray) ToServiceVolumes() []types.ServiceVolumeConfig {
	volumes := []types.ServiceVolumeConfig{}

	for _, path := range *p {

		volumeType := "volume"
		if strings.Contains(path.Path, "/") {
			volumeType = "bind"
		}

		volumes = append(volumes, types.ServiceVolumeConfig{
			Type:   volumeType,
			Source: path.Path,
			Target: path.ContainerPath,
		})
	}
	return volumes
}

func (c *CustomizationPostData) ToCompose() *codegen.ComposeApp {
	appStoreInfo := codegen.AppStoreInfo{
		// TODO
	}

	services := types.Services{
		{
			Name:        strings.ToLower(c.ContainerName),
			Image:       c.Image,
			NetworkMode: c.NetworkModel,
			Ports:       c.Ports.ToServicePorts(),
			Restart:     c.Restart,
			Volumes:     c.Volumes.ToServiceVolumes(),

			CPUShares: c.CPUShares,

			Extensions: map[string]interface{}{
				common.ComposeExtensionNameXCasaOS: appStoreInfo,
			},
		},
	}

	composeAppStoreInfo := codegen.ComposeAppStoreInfo{
		// TODO
	}

	compose := codegen.ComposeApp{
		Name:     strings.ToLower(c.ContainerName),
		Services: services,
		Extensions: map[string]interface{}{
			common.ComposeExtensionNameXCasaOS: composeAppStoreInfo,
		},
	}

	return &compose
}
