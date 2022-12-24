package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/compose-spec/compose-go/types"
	"github.com/samber/lo"
)

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

	appStoreInfo := &codegen.AppStoreInfo{
		Author:   a.Adaptor.Name,
		Category: a.Category[0],
		Container: codegen.ContainerStoreInfo{
			PortMap: a.Container.WebUI.HTTP,
			Index:   a.Container.WebUI.Path,
			Shell:   lo.If(a.Container.Shell != "", &a.Container.Shell).Else(nil),
			Envs: []codegen.EnvStoreInfo{
				{},
			},
			Ports:   []codegen.PortStoreInfo{},
			Volumes: []codegen.VolumeStoreInfo{},
		},
		Description:    langTextMap(a.Overview),
		Developer:      a.Developer.Name,
		Icon:           a.Icon,
		ScreenshotLink: a.Screenshots,
		Tagline:        langTextMap(a.Tagline),
		Thumbnail:      a.Thumbnail,
		Title:          langTextMap(a.Title),
	}

	return appStoreInfo
}

func (a *AppFile) ComposeAppStoreInfo() *codegen.ComposeAppStoreInfo {
	return nil
}

func (a *AppFile) ComposeApp() *types.Project {
	environment := make(map[string]string, len(a.Container.Envs))
	for _, env := range a.Container.Envs {
		environment[env.Key] = env.Value
	}

	// TODO: add volumes, ports, devices

	composeApp := &types.Project{
		Name:        a.Name,
		Environment: environment,
	}

	return composeApp
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

	return appFile, nil
}

func langTextMap(text string) map[string]string {
	return map[string]string{
		"en_US": text,
	}
}
