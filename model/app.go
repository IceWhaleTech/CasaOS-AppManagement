package model

import (
	"time"
)

type ServerAppListCollection struct {
	List      []ServerAppList `json:"list"`
	Recommend []ServerAppList `json:"recommend"`
	Community []ServerAppList `json:"community"`
}

type StateEnum int

const (
	StateEnumNotInstalled StateEnum = iota
	StateEnumInstalled
)

// @tiger - 对于用于出参的数据结构，静态信息（例如 title）和
//
//	动态信息（例如 state、query_count）应该划分到不同的数据结构中
//
//	这样的好处是
//	1 - 多次获取动态信息时可以减少出参复杂度，因为静态信息只获取一次就好
//	2 - 在未来的迭代中，可以降低维护成本（所有字段都展开放在一个层级维护成本略高）
//
//	另外，一些针对性字段，例如 Docker 相关的，可以用 map 来保存。
//	这样在未来增加多态 App，例如 Snap，不需要维护多个结构，或者一个结构保存不必要的字段
type ServerAppList struct {
	ID             uint      `gorm:"column:id;primary_key" json:"id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Tagline        string    `json:"tagline"`
	Tags           Strings   `gorm:"type:json" json:"tags"`
	Icon           string    `json:"icon"`
	ScreenshotLink Strings   `gorm:"type:json" json:"screenshot_link"`
	Category       string    `json:"category"`
	CategoryID     int       `json:"category_id"`
	CategoryFont   string    `json:"category_font"`
	PortMap        string    `json:"port_map"`
	ImageVersion   string    `json:"image_version"`
	Tip            string    `json:"tip"`
	Envs           EnvArray  `json:"envs"`
	Ports          PortArray `json:"ports"`
	Volumes        PathArray `json:"volumes"`
	Devices        PathArray `json:"devices"`
	NetworkModel   string    `json:"network_model"`
	Image          string    `json:"image"`
	Index          string    `json:"index"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	State          StateEnum `json:"state"`
	Author         string    `json:"author"`
	MinMemory      int       `json:"min_memory"`
	MinDisk        int       `json:"min_disk"`
	Thumbnail      string    `json:"thumbnail"`
	Healthy        string    `json:"healthy"`
	Plugins        Strings   `json:"plugins"`
	Origin         string    `json:"origin"`
	Type           int       `json:"type"`
	QueryCount     int       `json:"query_count"`
	Developer      string    `json:"developer"`
	HostName       string    `json:"host_name"`
	Privileged     bool      `json:"privileged"`
	CapAdd         Strings   `json:"cap_add"`
	Cmd            Strings   `json:"cmd"`
	Architectures  Strings   `json:"architectures"`
	LatestDigest   Strings   `json:"latest_digests"`
}

type MyAppList struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Icon       string `json:"icon"`
	State      string `json:"state"`
	CustomID   string `gorm:"column:custom_id;primary_key" json:"custom_id"`
	Index      string `json:"index"`
	Port       string `json:"port"`
	Slogan     string `json:"slogan"`
	Type       string `json:"type"`
	Image      string `json:"image"`
	Volumes    string `json:"volumes"`
	Latest     bool   `json:"latest"`
	Host       string `json:"host"`
	Protocol   string `json:"protocol"`
	Created    int64  `json:"created"`
	AppStoreID uint   `json:"appstore_id"`
}

type Ports struct {
	ContainerPort uint   `json:"container_port"`
	CommendPort   int    `json:"commend_port"`
	Desc          string `json:"desc"`
	Type          int    `json:"type"` //  1:必选 2:可选 3:默认值不必显示 4:系统处理  5:container内容也可编辑
}

type Volume struct {
	ContainerPath string `json:"container_path"`
	Path          string `json:"path"`
	Desc          string `json:"desc"`
	Type          int    `json:"type"` //  1:必选 2:可选 3:默认值不必显示 4:系统处理   5:container内容也可编辑
}

type Envs struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Desc  string `json:"desc"`
	Type  int    `json:"type"` //  1:必选 2:可选 3:默认值不必显示 4:系统处理 5:container内容也可编辑
}

type Devices struct {
	ContainerPath string `json:"container_path"`
	Path          string `json:"path"`
	Desc          string `json:"desc"`
	Type          int    `json:"type"` //  1:必选 2:可选 3:默认值不必显示 4:系统处理 5:container内容也可编辑
}

type Strings []string

type MapStrings []map[string]string
