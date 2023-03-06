package main

import (
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/compose-spec/compose-go/types"
	jsoniter "github.com/json-iterator/go"
	"github.com/samber/lo"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// https://raw.githubusercontent.com/IceWhaleTech/CasaOS-AppStore/main/Apps/FileBrowser/appfile.json
type AppFile struct {
	ID          int      `json:"id"`
	Version     string   `json:"version"`
	Title       string   `json:"title"`
	Name        string   `json:"name"`
	Icon        string   `json:"icon"`
	Tagline     string   `json:"tagline"`
	Overview    string   `json:"overview"`
	Thumbnail   string   `json:"thumbnail"`
	Type        int      `json:"type"` // 0:官方  1:推荐  2:社区
	Screenshots []string `json:"screenshots"`
	Category    []string `json:"category"`
	Adaptor     struct {
		Name       string `json:"name"`
		Website    string `json:"website"`
		DonateText string `json:"donate_text"`
		DonateLink string `json:"donate_link"`
	} `json:"adaptor"`
	Developer struct {
		Name       string `json:"name"`
		Website    string `json:"website"`
		DonateText string `json:"donate_text"`
		DonateLink string `json:"donate_link"`
	} `json:"developer"`
	Support   string `json:"support"`
	Website   string `json:"website"`
	Container struct {
		Image        string `json:"image"`
		Shell        string `json:"shell"`
		Privileged   bool   `json:"privileged"`
		NetworkModel string `json:"network_model"`
		Healthy      string `json:"healthy"`
		WebUI        struct {
			HTTP string `json:"http"`
			Path string `json:"path"`
		} `json:"web_ui"`
		Envs []struct {
			Key          string `json:"key"`
			Value        string `json:"value"`
			Configurable string `json:"configurable"`
			Description  string `json:"description"`
		} `json:"envs"`
		Ports []struct {
			Container    string `json:"container"`
			Host         string `json:"host"`
			Type         string `json:"type"`
			Allocation   string `json:"allocation"`
			Configurable string `json:"configurable"`
			Description  string `json:"description"`
		} `json:"ports"`
		Volumes []struct {
			Container    string `json:"container"`
			Host         string `json:"host"`
			Mode         string `json:"mode"`
			Allocation   string `json:"allocation"`
			Configurable string `json:"configurable"`
			Description  string `json:"description"`
		} `json:"volumes"`
		Devices []struct {
			Container    string `json:"container"`
			Host         string `json:"host"`
			Allocation   string `json:"allocation"`
			Configurable string `json:"configurable"`
			Description  string `json:"description"`
		} `json:"devices"`
		Constraints struct {
			MinMemory  int `json:"min_memory"`
			MinStorage int `json:"min_storage"`
		} `json:"constraints"`
		RestartPolicy string        `json:"restart_policy"`
		Sysctls       []interface{} `json:"sysctls"`
		CapAdd        struct{}      `json:"cap_add"`
		Labels        []interface{} `json:"labels"`
	} `json:"container"`
	Abilities struct {
		Notification   bool `json:"notification"`
		Widgets        bool `json:"widgets"`
		Authentication bool `json:"authentication"`
		Search         bool `json:"search"`
		Upnp           bool `json:"upnp"`
	} `json:"abilities"`
	Tips struct {
		BeforeInstall []struct {
			Content string `json:"content"`
			Value   string `json:"value" `
		} `json:"before_install"`
	} `json:"tips,omitempty"`
	Changelog struct {
		LatestUpdates string `json:"latest_updates"`
		URL           string `json:"url"`
	} `json:"changelog"`
	LatestUpdateDate string   `json:"latest_update_date"`
	CMD              []string `json:"cmd"`
}

func (a *AppFile) AppStoreInfo() *codegen.AppStoreInfo {
	envs := make([]codegen.EnvStoreInfo, len(a.Container.Envs))
	for i, env := range a.Container.Envs {
		envs[i] = codegen.EnvStoreInfo{
			Container:    env.Key,
			Description:  langTextMap(env.Description),
			Configurable: codegen.Configurable(env.Configurable),
		}
	}

	ports := make([]codegen.PortStoreInfo, len(a.Container.Ports))
	for i, port := range a.Container.Ports {
		ports[i] = codegen.PortStoreInfo{
			Container:    port.Container,
			Protocol:     codegen.PortStoreInfoProtocol(strings.ToLower(port.Type)),
			Description:  langTextMap(port.Description),
			Configurable: codegen.Configurable(port.Configurable),
		}
	}

	volumes := make([]codegen.VolumeStoreInfo, len(a.Container.Volumes))
	for i, volume := range a.Container.Volumes {
		volumes[i] = codegen.VolumeStoreInfo{
			Container:    volume.Container,
			Description:  langTextMap(volume.Description),
			Configurable: codegen.Configurable(volume.Configurable),
		}
	}

	tipsBeforeInstall := make([]codegen.Tip, len(a.Tips.BeforeInstall))
	for i, tip := range a.Tips.BeforeInstall {
		tipsBeforeInstall[i] = codegen.Tip{
			Content: langTextMap(tip.Content),
			Value:   lo.If(tip.Value != "", &a.Tips.BeforeInstall[i].Value).Else(nil),
		}
	}

	appStoreInfo := &codegen.AppStoreInfo{
		Author:         a.Adaptor.Name,
		Category:       a.Category[0],
		Description:    langTextMap(a.Overview),
		Developer:      a.Developer.Name,
		Icon:           a.Icon,
		ScreenshotLink: a.Screenshots,
		Tagline:        langTextMap(a.Tagline),
		Thumbnail:      a.Thumbnail,
		Title:          langTextMap(a.Title),
		Tips: codegen.TipsStoreInfo{
			BeforeInstall: tipsBeforeInstall,
		},
		Container: codegen.ContainerStoreInfo{
			PortMap: a.Container.WebUI.HTTP,
			Index:   a.Container.WebUI.Path,
			Shell:   lo.If(a.Container.Shell != "", &a.Container.Shell).Else(nil),
			Envs:    envs,
			Ports:   ports,
			Volumes: volumes,
		},
	}

	return appStoreInfo
}

func (a *AppFile) ComposeAppStoreInfo() *codegen.ComposeAppStoreInfo {
	// get tag of a docker image
	tag := "TBD"
	imageAndTag := strings.Split(a.Container.Image, ":")
	if len(imageAndTag) > 1 {
		tag = imageAndTag[1]
	}

	architectures := []string{"amd64"}
	_architectures, err := docker.GetArchitectures(a.Container.Image, false)
	if err == nil {
		architectures = _architectures
	}

	return &codegen.ComposeAppStoreInfo{
		MainApp:       &a.Name,
		Version:       &tag,
		Architectures: &architectures,
	}
}

func (a *AppFile) ComposeApp() *service.ComposeApp {
	environment := make(map[string]*string, len(a.Container.Envs))
	for i, env := range a.Container.Envs {
		environment[env.Key] = &a.Container.Envs[i].Value
	}

	ports := make([]types.ServicePortConfig, len(a.Container.Ports))
	for i, port := range a.Container.Ports {
		target, err := strconv.Atoi(port.Container)
		if err != nil {
			continue
		}

		ports[i] = types.ServicePortConfig{
			Target:    uint32(target),
			Published: port.Host,
			Protocol:  port.Type,
		}
	}

	volumes := make([]types.ServiceVolumeConfig, len(a.Container.Volumes))
	for i, volume := range a.Container.Volumes {
		volumes[i] = types.ServiceVolumeConfig{
			Type:     "bind",
			Target:   volume.Container,
			Source:   volume.Host,
			ReadOnly: strings.ToLower(volume.Mode) == "ro",
		}
	}

	devices := make([]string, len(a.Container.Devices))
	for i, device := range a.Container.Devices {
		devices[i] = device.Host + ":" + device.Container
	}

	services := []types.ServiceConfig{{
		Name:           a.Name,
		Image:          a.Container.Image,
		Privileged:     a.Container.Privileged,
		NetworkMode:    a.Container.NetworkModel,
		Environment:    environment,
		Ports:          ports,
		Volumes:        volumes,
		Devices:        devices,
		MemReservation: types.UnitBytes(a.Container.Constraints.MinMemory * 1024 * 1024),
		Restart:        a.Container.RestartPolicy,
		Extensions: map[string]interface{}{
			common.ComposeExtensionNameXCasaOS: a.AppStoreInfo(),
		},
	}}

	composeApp := (service.ComposeApp)(types.Project{
		Name:     a.Name,
		Services: services,
		Extensions: map[string]interface{}{
			common.ComposeExtensionNameXCasaOS: a.ComposeAppStoreInfo(),
		},
	})

	return &composeApp
}

func NewAppFile(path string) (*AppFile, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	appFile := &AppFile{}

	if err = json.Unmarshal(data, appFile); err != nil {
		return nil, err
	}

	appFile.Name = service.Standardize(appFile.Name)

	return appFile, nil
}

func langTextMap(text string) map[string]string {
	return map[string]string{
		"en_US": text,
	}
}
