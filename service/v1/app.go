package v1

import (
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
)

const (
	V1LabelName = "name"
	V1LabelIcon = "icon"
)

func GetCustomizationPostData(info types.ContainerJSON) model.CustomizationPostData {
	var port model.PortArray

	for k, v := range info.HostConfig.PortBindings {
		temp := model.PortMap{
			CommendPort:   v[0].HostPort,
			ContainerPort: k.Port(),

			Protocol: strings.ToLower(k.Proto()),
		}
		port = append(port, temp)
	}

	var envs model.EnvArray

	showENV := info.Config.Labels["show_env"]
	showENVList := strings.Split(showENV, ",")
	showENVMap := make(map[string]string)
	if len(showENVList) > 0 && showENVList[0] != "" {
		for _, name := range showENVList {
			showENVMap[name] = "1"
		}
	}
	for _, v := range info.Config.Env {
		env := strings.SplitN(v, "=", 2)
		if len(showENVList) > 0 && info.Config.Labels["origin"] != "local" {
			if _, ok := showENVMap[env[0]]; ok {
				temp := model.Env{Name: env[0], Value: env[1]}
				envs = append(envs, temp)
			}
		} else {
			temp := model.Env{Name: env[0], Value: env[1]}
			envs = append(envs, temp)
		}
	}

	var vol model.PathArray

	for i := 0; i < len(info.Mounts); i++ {
		temp := model.PathMap{
			Path:          strings.ReplaceAll(info.Mounts[i].Source, "$AppID", info.Name),
			ContainerPath: info.Mounts[i].Destination,
		}
		vol = append(vol, temp)
	}
	var driver model.PathArray

	for _, v := range info.HostConfig.Resources.Devices {
		temp := model.PathMap{
			Path:          v.PathOnHost,
			ContainerPath: v.PathInContainer,
		}
		driver = append(driver, temp)
	}

	name := AppName(&info)
	if len(name) == 0 {
		name = strings.ReplaceAll(info.Name, "/", "")
	}

	var appStoreID uint
	if appStoreIDStr, ok := info.Config.Labels[common.ContainerLabelV1AppStoreID]; ok {
		_appStoreID, err := strconv.Atoi(appStoreIDStr)
		if err != nil {
			logger.Error("error when converting appStoreID from string to int", zap.Error(err), zap.String("appStoreIDStr", appStoreIDStr))
		}

		if _appStoreID > 0 {
			appStoreID = uint(_appStoreID)
		}
	}

	m := model.CustomizationPostData{
		AppStoreID:    appStoreID,
		CapAdd:        info.HostConfig.CapAdd,
		Cmd:           info.Config.Cmd,
		ContainerName: strings.ReplaceAll(info.Name, "/", ""),
		CPUShares:     info.HostConfig.CPUShares,
		CustomID:      info.Config.Labels["custom_id"],
		Description:   info.Config.Labels["desc"],
		Devices:       driver,
		EnableUPNP:    false,
		Envs:          envs,
		Host:          info.Config.Labels["host"],
		HostName:      info.Config.Hostname,
		Icon:          AppIcon(&info),
		Image:         info.Config.Image,
		Index:         info.Config.Labels["index"],
		Label:         name,
		Memory:        info.HostConfig.Memory >> 20,
		NetworkModel:  string(info.HostConfig.NetworkMode),
		Origin:        info.Config.Labels["origin"],
		PortMap:       info.Config.Labels["web"],
		Ports:         port,
		Position:      false,
		Privileged:    info.HostConfig.Privileged,
		Protocol:      info.Config.Labels["protocol"],
		Restart:       info.HostConfig.RestartPolicy.Name,
		Volumes:       vol,
	}

	if len(m.Origin) == 0 {
		m.Origin = "local"
	}

	if len(m.CustomID) == 0 {
		m.CustomID = uuid.NewV4().String()
	}

	if m.Protocol == "" {
		m.Protocol = "http"
	}

	return m
}

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
