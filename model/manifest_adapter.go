package model

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/compose-spec/compose-go/types"
	"github.com/samber/lo"
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

func (p *PortArray) ToPortStoreInfo() []codegen.PortStoreInfo {
	return lo.Map(*p, func(p PortMap, i int) codegen.PortStoreInfo {
		return codegen.PortStoreInfo{
			Container:   p.ContainerPort,
			Description: map[string]string{common.DefaultLanguage: p.Desc},
		}
	})
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

func (p *PathArray) ToSlice() []string {
	return lo.Map(*p, func(p PathMap, i int) string {
		return fmt.Sprintf("%s:%s", p.Path, p.ContainerPath)
	})
}

func (p *PathArray) ToDeviceStoreInfoList() []codegen.DeviceStoreInfo {
	return lo.Map(*p, func(p PathMap, i int) codegen.DeviceStoreInfo {
		return codegen.DeviceStoreInfo{
			Container:   &p.ContainerPath,
			Description: &map[string]string{common.DefaultLanguage: p.Desc},
		}
	})
}

func (p *PathArray) ToVolumeStoreInfoList() []codegen.VolumeStoreInfo {
	return lo.Map(*p, func(p PathMap, i int) codegen.VolumeStoreInfo {
		return codegen.VolumeStoreInfo{
			Container:   p.ContainerPath,
			Description: map[string]string{common.DefaultLanguage: p.Desc},
		}
	})
}

func (ea *EnvArray) ToMappingWithEquals() types.MappingWithEquals {
	return lo.SliceToMap(*ea, func(e Env) (string, *string) {
		return e.Name, &e.Value
	})
}

func (ea *EnvArray) ToEnvStoreInfoList() []codegen.EnvStoreInfo {
	return lo.Map(*ea, func(e Env, i int) codegen.EnvStoreInfo {
		return codegen.EnvStoreInfo{
			Container:   e.Name,
			Description: map[string]string{common.DefaultLanguage: e.Desc},
		}
	})
}

func (c *CustomizationPostData) AppStoreInfo() codegen.AppStoreInfo {
	return codegen.AppStoreInfo{
		Devices: c.Devices.ToDeviceStoreInfoList(),
		Envs:    c.Envs.ToEnvStoreInfoList(),
		Ports:   c.Ports.ToPortStoreInfo(),
		Volumes: c.Volumes.ToVolumeStoreInfoList(),
	}
}

func (c *CustomizationPostData) ComposeAppStoreInfo() codegen.ComposeAppStoreInfo {
	currentArchitecture := currentArchitecture()
	name := strings.ToLower(c.ContainerName)

	return codegen.ComposeAppStoreInfo{
		Architectures: &[]string{currentArchitecture},
		Main:          &name,
		Author:        "custom",
		Description: map[string]string{
			common.DefaultLanguage: c.Description,
		},
		Developer: "unknown",
		Icon:      c.Icon,
		Title: map[string]string{
			common.DefaultLanguage: c.Label,
		},
		Index:   c.Index,
		PortMap: c.PortMap,
	}
}

func (c *CustomizationPostData) Services() types.Services {
	return types.Services{
		{
			CapAdd:      c.CapAdd,
			Command:     c.Cmd,
			CPUShares:   c.CPUShares,
			Devices:     c.Devices.ToSlice(),
			Environment: c.Envs.ToMappingWithEquals(),
			Image:       c.Image,
			Name:        strings.ToLower(c.ContainerName),
			NetworkMode: c.NetworkModel,
			Ports:       c.Ports.ToServicePorts(),
			Privileged:  c.Privileged,
			Restart:     c.Restart,
			Volumes:     c.Volumes.ToServiceVolumes(),

			Extensions: map[string]interface{}{
				common.ComposeExtensionNameXCasaOS: c.AppStoreInfo(),
			},
		},
	}
}

func (c *CustomizationPostData) Compose() *codegen.ComposeApp {
	return &codegen.ComposeApp{
		Name:     strings.ToLower(c.ContainerName),
		Services: c.Services(),
		Extensions: map[string]interface{}{
			common.ComposeExtensionNameXCasaOS: c.ComposeAppStoreInfo(),
		},
	}
}

func currentArchitecture() string {
	arch := runtime.GOARCH

	if arch == "arm" {
		arch = "arm-7"
	}

	return arch
}
