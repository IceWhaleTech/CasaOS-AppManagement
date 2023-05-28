package model

type TCPPorts struct {
	Desc          string `json:"desc"`
	ContainerPort int    `json:"container_port"`
}
type UDPPorts struct {
	Desc          string `json:"desc"`
	ContainerPort int    `json:"container_port"`
}

/*******************使用gorm支持json************************************/

type PortMap struct {
	ContainerPort string `json:"container"`
	CommendPort   string `json:"host"`
	Protocol      string `json:"protocol"`
	Desc          string `json:"desc"`
	Type          int    `json:"type"`
}

type PortArray []PortMap

/************************************************************************/

/*******************使用gorm支持json************************************/

type Env struct {
	Name  string `json:"container"`
	Value string `json:"host"`
	Desc  string `json:"desc"`
	Type  int    `json:"type"`
}

type EnvArray []Env

/************************************************************************/

/*******************使用gorm支持json************************************/

type PathMap struct {
	ContainerPath string `json:"container"`
	Path          string `json:"host"`
	Type          int    `json:"type"`
	Desc          string `json:"desc"`
}

type PathArray []PathMap

/************************************************************************/

//type PostData struct {
//	Envs       EnvArrey  `json:"envs,omitempty"`
//	Udp        PortArrey `json:"udp_ports"`
//	Tcp        PortArrey `json:"tcp_ports"`
//	Volumes    PathArrey `json:"volumes"`
//	Devices    PathArrey `json:"devices"`
//	Port       string    `json:"port,omitempty"`
//	PortMap    string    `json:"port_map"`
//	CpuShares  int64     `json:"cpu_shares,omitempty"`
//	Memory     int64     `json:"memory,omitempty"`
//	Restart    string    `json:"restart,omitempty"`
//	EnableUPNP bool      `json:"enable_upnp"`
//	Label      string    `json:"label"`
//	Position   bool      `json:"position"`
//}

type CustomizationPostData struct {
	ContainerName string    `json:"container_name"`
	CustomID      string    `json:"custom_id"`
	Origin        string    `json:"origin"`
	NetworkModel  string    `json:"network_model"`
	Index         string    `json:"index"`
	Icon          string    `json:"icon"`
	Image         string    `json:"image"`
	Envs          EnvArray  `json:"envs"`
	Ports         PortArray `json:"ports"`
	Volumes       PathArray `json:"volumes"`
	Devices       PathArray `json:"devices"`
	// Port         string    `json:"port,omitempty"`
	PortMap     string   `json:"port_map"`
	CPUShares   int64    `json:"cpu_shares"`
	Memory      int64    `json:"memory"`
	Restart     string   `json:"restart"`
	EnableUPNP  bool     `json:"enable_upnp"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Position    bool     `json:"position"`
	HostName    string   `json:"host_name"`
	Privileged  bool     `json:"privileged"`
	CapAdd      []string `json:"cap_add"`
	Cmd         []string `json:"cmd"`
	Protocol    string   `json:"protocol"`
	Host        string   `json:"host"`
	AppStoreID  uint     `json:"appstore_id"`
}
