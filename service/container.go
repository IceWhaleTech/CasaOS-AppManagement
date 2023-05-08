package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.uber.org/zap"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/utils/envHelper"
	v1 "github.com/IceWhaleTech/CasaOS-AppManagement/service/v1"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-Common/utils/random"
	timeutils "github.com/IceWhaleTech/CasaOS-Common/utils/time"

	//"github.com/containerd/containerd/oci"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	client2 "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

var (
	dataStats = &sync.Map{}
	isFinish  bool

	NewVersionApp map[string]string // TODO - make use of NewVersionApp map
)

type DockerService interface {
	// image
	IsExistImage(imageName string) bool
	PullImage(ctx context.Context, imageName string) error
	PullLatestImage(ctx context.Context, imageName string) (bool, error)
	RemoveImage(name string) error

	// container
	CheckContainerHealth(id string) (bool, error)
	CreateContainer(m model.CustomizationPostData, id string) (containerID string, err error)
	CreateContainerShellSession(container, row, col string) (hr types.HijackedResponse, err error)
	DescribeContainer(ctx context.Context, name string) (*types.ContainerJSON, error)
	GetContainer(id string) (types.Container, error)
	GetContainerAppList(name, image, state *string) (*[]model.MyAppList, *[]model.MyAppList)
	GetContainerByName(name string) (*types.Container, error)
	GetContainerLog(name string) ([]byte, error)
	GetContainerStats() []model.DockerStatsModel
	RecreateContainer(ctx context.Context, id string, pull bool, force bool) error
	RemoveContainer(name string, update bool) error
	RenameContainer(name, id string) (err error)
	StartContainer(name string) error
	StopContainer(id string) error

	// network
	GetNetworkList() []types.NetworkResource

	// docker server
	GetServerInfo() (types.Info, error)
}

type dockerService struct{}

// FIXME - should use WebSocket or SocketIO instead of HTTP polling (tiger)
func getContainerStats() {
	cli, err := client2.NewClientWithOpts(client2.FromEnv, client2.WithAPIVersionNegotiation())
	if err != nil {
		return
	}
	defer cli.Close()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		logger.Error("Failed to get container_list", zap.Any("err", err))
	}
	for i := 0; i < 100; i++ {
		if i%10 == 0 {
			containers, err = cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
			if err != nil {
				logger.Error("Failed to get container_list", zap.Any("err", err))
				continue
			}
		}
		if config.CasaOSGlobalVariables.AppChange {
			config.CasaOSGlobalVariables.AppChange = false
			dataStats.Range(func(key, value interface{}) bool {
				dataStats.Delete(key)
				return true
			})
		}

		var temp sync.Map
		var wg sync.WaitGroup
		for _, v := range containers {
			if v.State != "running" {
				continue
			}
			wg.Add(1)
			go func(v types.Container, i int) {
				defer wg.Done()
				stats, err := cli.ContainerStatsOneShot(ctx, v.ID)
				if err != nil {
					return
				}
				decoder := json.NewDecoder(stats.Body)

				// data
				var data interface{}
				if err := decoder.Decode(&data); err == io.EOF {
					return
				}
				m, _ := dataStats.Load(v.ID)
				dockerStats := model.DockerStatsModel{}
				if m != nil {
					dockerStats.Previous = m.(model.DockerStatsModel).Data
				}

				// icon
				if icon, ok := v.Labels[v1.V1LabelIcon]; ok {
					dockerStats.Icon = icon
				}

				dockerStats.Data = data
				dockerStats.Title = strings.ReplaceAll(v.Names[0], "/", "")

				// @tiger - 不建议直接把依赖的数据结构封装返回。
				//          如果依赖的数据结构有变化，应该在这里适配或者保存，这样更加对客户端负责
				temp.Store(v.ID, dockerStats)
				if i == 99 {
					stats.Body.Close()
				}
			}(v, i)
		}
		wg.Wait()
		dataStats = &temp
		isFinish = true

		time.Sleep(time.Second * 1)
	}
	isFinish = false
	cancel()
}

func (ds *dockerService) GetContainerStats() []model.DockerStatsModel {
	stream := true
	for !isFinish {
		if stream {
			stream = false
			go getContainerStats()
		}
		runtime.Gosched()
	}
	list := []model.DockerStatsModel{}

	dataStats.Range(func(key, value interface{}) bool {
		list = append(list, value.(model.DockerStatsModel))
		return true
	})
	return list
}

func (ds *dockerService) CheckContainerHealth(id string) (bool, error) {
	container, err := ds.GetContainer(id)
	if err != nil {
		logger.Error("failed to get container by id", zap.Error(err), zap.String("id", id))
		return false, err
	}

	if webUIPort, ok := container.Labels["web"]; ok {
		index := container.Labels["index"]
		url := fmt.Sprintf("http://%s:%s/%s", common.Localhost, webUIPort, strings.TrimLeft(index, "/"))

		logger.Info("checking container health at the specified web port...", zap.Any("name", container.Names), zap.String("id", id), zap.Any("url", url))
		client := resty.New()
		client.SetTimeout(30 * time.Second)
		client.SetHeader("Accept", "text/html")
		response, err := client.R().Get(url)
		if err != nil {
			logger.Error("failed to check container health", zap.Error(err), zap.Any("name", container.Names), zap.String("id", id))
			return false, err
		}
		if response.StatusCode() == http.StatusOK || response.StatusCode() == http.StatusUnauthorized {
			return true, nil
		}
		// response, err := httpUtil.GetWithHeader(url, 30*time.Second, map[string]string{
		// 	echo.HeaderAccept: echo.MIMETextHTML, // emulate a browser
		// })
		// if err != nil {
		// 	logger.Error("failed to check container health", zap.Error(err), zap.Any("name", container.Names), zap.String("id", id))
		// 	return false, err
		// }

		// if (response.StatusCode == http.StatusUnauthorized) || // we treat Unauthorized as a success because it means the container is up and running
		// 	(response.StatusCode >= 200 && response.StatusCode < 300) {
		// 	logger.Info("container health check passed at the specified web port", zap.Any("name", container.Names), zap.String("id", id), zap.Any("url", url))
		// 	return true, nil
		// }

		logger.Error("container health check failed at the specified web port", zap.Any("name", container.Names), zap.String("id", id), zap.Any("url", url), zap.String("status", fmt.Sprint(response.StatusCode())))
		return false, errors.New(fmt.Sprint(response.StatusCode()))
	}

	logger.Error("container health check failed, no web port specified", zap.Any("name", container.Names), zap.String("id", id))
	return false, errors.New("no web port")
}

// 获取我的应用列表
func (ds *dockerService) GetContainer(id string) (types.Container, error) {
	// 获取docker应用
	cli, err := client2.NewClientWithOpts(client2.FromEnv, client2.WithAPIVersionNegotiation())
	if err != nil {
		logger.Error("Failed to init client", zap.Any("err", err))
		return types.Container{}, err
	}
	defer cli.Close()

	filters := filters.NewArgs()
	filters.Add("id", id)
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		logger.Error("Failed to get container_list", zap.Any("err", err))
		return types.Container{}, err
	}

	if len(containers) > 0 {
		return containers[0], nil
	}
	return types.Container{}, nil
}

// 获取我的应用列表
func (ds *dockerService) GetContainerAppList(name, image, state *string) (*[]model.MyAppList, *[]model.MyAppList) {
	cli, err := client2.NewClientWithOpts(client2.FromEnv, client2.WithAPIVersionNegotiation(), client2.WithTimeout(time.Second*5))
	if err != nil {
		logger.Error("Failed to init client", zap.Any("err", err))
	}
	defer cli.Close()
	// fts := filters.NewArgs()
	// fts.Add("label", "casaos=casaos")
	// fts.Add("label", "casaos")
	// fts.Add("casaos", "casaos")
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		logger.Error("Failed to get container_list", zap.Any("err", err))
	}
	// 获取本地数据库应用

	localApps := []model.MyAppList{}

	casaOSApps := []model.MyAppList{}

	for i, m := range containers {

		if name != nil && len(*name) > 0 {
			if !lo.ContainsBy(m.Names, func(n string) bool { return strings.Contains(n, *name) }) {
				continue
			}
		}

		if image != nil && len(*image) > 0 {
			if !strings.HasPrefix(m.Image, *image) {
				continue
			}
		}

		if state != nil && len(*state) > 0 {
			if m.State != *state {
				continue
			}
		}

		if m.Labels["casaos"] == "casaos" {

			_, newVersion := NewVersionApp[m.ID]
			name := strings.ReplaceAll(m.Names[0], "/", "")
			icon := m.Labels["icon"]
			if len(m.Labels["name"]) > 0 {
				name = m.Labels["name"]
			}
			if m.Labels["origin"] == "system" {
				name = strings.Split(m.Image, ":")[0]
				if len(strings.Split(name, "/")) > 1 {
					icon = "https://icon.casaos.io/main/all/" + strings.Split(name, "/")[1] + ".png"
				}
			}

			casaOSApp := model.MyAppList{
				Name:       name,
				Icon:       icon,
				State:      m.State,
				CustomID:   m.Labels["custom_id"],
				ID:         m.ID,
				Port:       m.Labels["web"],
				Index:      m.Labels["index"],
				Image:      m.Image,
				Latest:     newVersion,
				Host:       m.Labels["host"],
				Protocol:   m.Labels["protocol"],
				Created:    m.Created,
				AppStoreID: getV1AppStoreID(&containers[i]),
			}

			casaOSApps = append(casaOSApps, casaOSApp)
		} else {
			localApp := model.MyAppList{
				Name:     strings.ReplaceAll(m.Names[0], "/", ""),
				Icon:     "",
				State:    m.State,
				CustomID: m.ID,
				ID:       m.ID,
				Port:     "",
				Latest:   false,
				Host:     "",
				Protocol: "",
				Image:    m.Image,
				Created:  m.Created,
			}

			localApps = append(localApps, localApp)
		}
	}

	return &casaOSApps, &localApps
}

func (ds *dockerService) CreateContainerShellSession(container, row, col string) (types.HijackedResponse, error) {
	cli, err := client2.NewClientWithOpts(client2.FromEnv, client2.WithAPIVersionNegotiation())
	if err != nil {
		return types.HijackedResponse{}, err
	}

	ctx := context.Background()
	ir, err := cli.ContainerExecCreate(ctx, container, types.ExecConfig{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env:          []string{"COLUMNS=" + col, "LINES=" + row},
		Cmd:          []string{"/bin/sh"},
		Tty:          true,
	})
	if err != nil {
		return types.HijackedResponse{}, err
	}

	return cli.ContainerExecAttach(ctx, ir.ID, types.ExecStartCheck{Detach: false, Tty: true})
}

// 正式内容

// param imageName 镜像名称
// param containerDbId 数据库的id
// param port 容器内部主端口
// param mapPort 容器主端口映射到外部的端口
// param tcp 容器其他tcp端口
// param udp 容器其他udp端口
func (ds *dockerService) CreateContainer(m model.CustomizationPostData, id string) (containerID string, err error) {
	if len(m.NetworkModel) == 0 {
		m.NetworkModel = "bridge"
	}

	cli, err := client2.NewClientWithOpts(client2.FromEnv, client2.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}

	defer cli.Close()
	ports := make(nat.PortSet)
	portMaps := make(nat.PortMap)

	for _, portMap := range m.Ports {
		protocol := strings.ToLower(portMap.Protocol)

		if !lo.Contains([]string{"tcp", "udp", "both"}, protocol) {
			message := "unknown protocol"
			logger.Error(message, zap.String("protocol", protocol))
			return "", errors.New(message)
		}

		protocols := strings.Replace(protocol, "both", "tcp,udp", -1)
		for _, p := range strings.Split(protocols, ",") {
			tContainer, _ := strconv.Atoi(portMap.ContainerPort)
			if tContainer > 0 {
				ports[nat.Port(portMap.ContainerPort+"/"+p)] = struct{}{}
				if m.NetworkModel != "host" {
					portMaps[nat.Port(portMap.ContainerPort+"/"+p)] = []nat.PortBinding{{HostPort: portMap.CommendPort}}
				}
			}
		}
	}

	var envArr []string

	showENV := []string{"casaos"}

	for _, e := range m.Envs {
		showENV = append(showENV, e.Name)
		if strings.HasPrefix(e.Value, "$") {
			systemTimeZoneName := timeutils.GetSystemTimeZoneName()
			envArr = append(envArr, e.Name+"="+envHelper.ReplaceDefaultENV(e.Value, systemTimeZoneName))
			continue
		}
		if len(e.Value) > 0 {
			if e.Value == "port_map" {
				envArr = append(envArr, e.Name+"="+m.PortMap)
				continue
			}
			envArr = append(envArr, e.Name+"="+e.Value)
		}
	}

	res := container.Resources{}
	if m.CPUShares > 0 {
		res.CPUShares = m.CPUShares
	}
	if m.Memory > 0 {
		res.Memory = m.Memory << 20
	}
	for _, p := range m.Devices {
		if len(p.Path) > 0 {
			res.Devices = append(res.Devices, container.DeviceMapping{PathOnHost: p.Path, PathInContainer: p.ContainerPath, CgroupPermissions: "rwm"})
		}
	}
	hostConfingBind := []string{}
	// volumes bind
	volumes := []mount.Mount{}
	for _, v := range m.Volumes {
		path := v.Path
		if len(path) == 0 {
			path = docker.GetDir(m.Label, v.Path)
			if len(path) == 0 {
				continue
			}
		}
		path = strings.ReplaceAll(path, "$AppID", m.Label)
		// reg1 := regexp.MustCompile(`([^<>/\\\|:""\*\?]+\.\w+$)`)
		// result1 := reg1.FindAllStringSubmatch(path, -1)
		// if len(result1) == 0 {
		err = file.IsNotExistMkDir(path)
		if err != nil {
			logger.Error("Failed to create a folder", zap.Any("err", err))
			continue
		}
		//}
		//  else {
		// 	err = file.IsNotExistCreateFile(path)
		// 	if err != nil {
		// 		ds.log.Error("mkdir error", err)
		// 		continue
		// 	}
		// }

		volumes = append(volumes, mount.Mount{
			Type:   mount.TypeBind,
			Source: path,
			Target: v.ContainerPath,
		})

		hostConfingBind = append(hostConfingBind, v.Path+":"+v.ContainerPath)
	}

	rp := container.RestartPolicy{}

	if len(m.Restart) > 0 {
		rp.Name = m.Restart
	}
	// healthTest := []string{}
	// if len(port) > 0 {
	// 	healthTest = []string{"CMD-SHELL", "curl -f http://localhost:" + port + m.Index + " || exit 1"}
	// }

	// health := &container.HealthConfig{
	// 	Test:        healthTest,
	// 	StartPeriod: 0,
	// 	Retries:     1000,
	// }
	// fmt.Print(health)
	if len(m.HostName) == 0 {
		m.HostName = m.Label
	}

	info, err := cli.ContainerInspect(context.Background(), id)
	hostConfig := &container.HostConfig{}
	config := &container.Config{}
	config.Labels = map[string]string{}
	if err == nil {
		// info.HostConfig = &container.HostConfig{}
		// info.Config = &container.Config{}
		// info.NetworkSettings = &types.NetworkSettings{}
		hostConfig = info.HostConfig
		config = info.Config
		if config.Labels["casaos"] == "casaos" {
			config.Cmd = m.Cmd
			config.Image = m.Image
			config.Env = envArr
			config.Hostname = m.HostName
			config.ExposedPorts = ports
		}
	} else {
		config.Cmd = m.Cmd
		config.Image = m.Image
		config.Env = envArr
		config.Hostname = m.HostName
		config.ExposedPorts = ports
	}

	config.Labels["origin"] = m.Origin
	config.Labels["casaos"] = "casaos"
	config.Labels["web"] = m.PortMap
	config.Labels["icon"] = m.Icon
	config.Labels["desc"] = m.Description
	config.Labels["index"] = m.Index
	config.Labels["custom_id"] = m.CustomID
	config.Labels["show_env"] = strings.Join(showENV, ",")
	config.Labels["protocol"] = m.Protocol
	config.Labels["host"] = m.Host
	config.Labels["name"] = m.Label
	config.Labels[common.ContainerLabelV1AppStoreID] = strconv.Itoa((int)(m.AppStoreID))
	// container, err := cli.ContainerCreate(context.Background(), info.Config, info.HostConfig, &network.NetworkingConfig{info.NetworkSettings.Networks}, nil, info.Name)

	hostConfig.Mounts = volumes
	hostConfig.Binds = []string{}
	hostConfig.Privileged = m.Privileged
	hostConfig.CapAdd = m.CapAdd
	hostConfig.NetworkMode = container.NetworkMode(m.NetworkModel)
	hostConfig.RestartPolicy = rp
	hostConfig.Resources = res
	// hostConfig := &container.HostConfig{Resources: res, Mounts: volumes, RestartPolicy: rp, NetworkMode: , Privileged: m.Privileged, CapAdd: m.CapAdd}
	// if net != "host" {

	hostConfig.PortBindings = portMaps
	//}
	containerDb, err := cli.ContainerCreate(context.Background(),
		config,
		hostConfig,
		&network.NetworkingConfig{EndpointsConfig: map[string]*network.EndpointSettings{m.NetworkModel: {NetworkID: "", Aliases: []string{}}}},
		nil,
		m.ContainerName)
	if err != nil {
		return "", err
	}
	return containerDb.ID, err
}

func (ds *dockerService) RecreateContainer(ctx context.Context, id string, pull bool, force bool) error {
	containerInfo, err := docker.Container(ctx, id)
	if err != nil {
		return err
	}

	isImageUpdated := false
	if pull {
		imageName := docker.ImageName(containerInfo)
		if imageName != "" {
			_isImageUpdated, err := ds.PullLatestImage(ctx, imageName) // image update result will be included in ctx properties
			if err != nil {
				logger.Error("pull new image failed", zap.Error(err), zap.String("image", imageName))
			}
			isImageUpdated = _isImageUpdated
		}
	}

	if !force && !isImageUpdated {
		return nil
	}

	// Clone the old container
	var newID string
	if err := func() error {
		tempName := fmt.Sprintf("%s-%s", containerInfo.Name, random.RandomString(4, false))

		go PublishEventWrapper(ctx, common.EventTypeContainerCreateBegin, map[string]string{
			common.PropertyTypeContainerName.Name: tempName,
		})

		defer PublishEventWrapper(ctx, common.EventTypeContainerCreateEnd, map[string]string{
			common.PropertyTypeContainerID.Name:   newID,
			common.PropertyTypeContainerName.Name: tempName,
		})

		_newID, err := docker.CloneContainer(ctx, id, tempName)
		if err != nil {
			go PublishEventWrapper(ctx, common.EventTypeContainerCreateError, map[string]string{
				common.PropertyTypeContainerName.Name: tempName,
				common.PropertyTypeMessage.Name:       err.Error(),
			})
			return err
		}
		newID = _newID

		return nil
	}(); err != nil {
		return err
	}

	// stop old container if it is running
	if containerInfo.State.Running {
		if err := func() error {
			go PublishEventWrapper(ctx, common.EventTypeContainerStopBegin, map[string]string{
				common.PropertyTypeContainerID.Name: id,
			})

			defer PublishEventWrapper(ctx, common.EventTypeContainerStopEnd, map[string]string{
				common.PropertyTypeContainerID.Name: id,
			})

			if err := docker.StopContainer(ctx, id); err != nil {
				go PublishEventWrapper(ctx, common.EventTypeContainerStopError, map[string]string{
					common.PropertyTypeContainerID.Name: id,
					common.PropertyTypeMessage.Name:     err.Error(),
				})
				return err
			}
			return nil
		}(); err != nil {
			return err
		}
	}

	// start new container
	if err := func() error {
		go PublishEventWrapper(ctx, common.EventTypeContainerStartBegin, map[string]string{
			common.PropertyTypeContainerID.Name: newID,
		})

		defer PublishEventWrapper(ctx, common.EventTypeContainerStartEnd, map[string]string{
			common.PropertyTypeContainerID.Name: newID,
		})

		if err := docker.StartContainer(ctx, newID); err != nil {
			go PublishEventWrapper(ctx, common.EventTypeContainerStartError, map[string]string{
				common.PropertyTypeContainerID.Name: newID,
				common.PropertyTypeMessage.Name:     err.Error(),
			})
			return err
		}
		return nil
	}(); err != nil {
		// if failed to start new container and old container was running...
		if containerInfo.State.Running {
			// start the old container
			if err := func() error {
				go PublishEventWrapper(ctx, common.EventTypeContainerStartBegin, map[string]string{
					common.PropertyTypeContainerID.Name: id,
				})

				defer PublishEventWrapper(ctx, common.EventTypeContainerStartEnd, map[string]string{
					common.PropertyTypeContainerID.Name: id,
				})

				if err := docker.StartContainer(ctx, id); err != nil {
					go PublishEventWrapper(ctx, common.EventTypeContainerStartError, map[string]string{
						common.PropertyTypeContainerID.Name: id,
						common.PropertyTypeMessage.Name:     err.Error(),
					})
					return err
				}
				return nil
			}(); err != nil {
				return err
			}

			// remove the new container
			if err := func() error {
				go PublishEventWrapper(ctx, common.EventTypeContainerRemoveBegin, map[string]string{
					common.PropertyTypeContainerID.Name: newID,
				})

				defer PublishEventWrapper(ctx, common.EventTypeContainerRemoveEnd, map[string]string{
					common.PropertyTypeContainerID.Name: newID,
				})

				if err := docker.RemoveContainer(ctx, newID); err != nil {
					go PublishEventWrapper(ctx, common.EventTypeContainerRemoveError, map[string]string{
						common.PropertyTypeContainerID.Name: newID,
						common.PropertyTypeMessage.Name:     err.Error(),
					})
					return err
				}
				return nil
			}(); err != nil {
				return err
			}
		}
	}

	// remove the old container if new container started successfully
	if err := func() error {
		go PublishEventWrapper(ctx, common.EventTypeContainerRemoveBegin, map[string]string{
			common.PropertyTypeContainerID.Name: containerInfo.ID,
		})

		defer PublishEventWrapper(ctx, common.EventTypeContainerRemoveEnd, map[string]string{
			common.PropertyTypeContainerID.Name: containerInfo.ID,
		})

		if err := docker.RemoveContainer(ctx, containerInfo.ID); err != nil {
			go PublishEventWrapper(ctx, common.EventTypeContainerRemoveError, map[string]string{
				common.PropertyTypeContainerID.Name: containerInfo.ID,
				common.PropertyTypeMessage.Name:     err.Error(),
			})
			return err
		}
		return nil
	}(); err != nil {
		return err
	}

	// rename the new container
	if err := func() error {
		go PublishEventWrapper(ctx, common.EventTypeContainerRenameBegin, map[string]string{
			common.PropertyTypeContainerID.Name:   newID,
			common.PropertyTypeContainerName.Name: containerInfo.Name,
		})

		defer PublishEventWrapper(ctx, common.EventTypeContainerRenameEnd, map[string]string{
			common.PropertyTypeContainerID.Name:   newID,
			common.PropertyTypeContainerName.Name: containerInfo.Name,
		})

		if err := docker.RenameContainer(ctx, newID, containerInfo.Name); err != nil {
			go PublishEventWrapper(ctx, common.EventTypeContainerRenameError, map[string]string{
				common.PropertyTypeContainerID.Name:   newID,
				common.PropertyTypeContainerName.Name: containerInfo.Name,
				common.PropertyTypeMessage.Name:       err.Error(),
			})

			return err
		}
		return nil
	}(); err != nil {
		return err
	}

	return nil
}

// 删除容器
func (ds *dockerService) RemoveContainer(name string, update bool) error {
	ctx := context.Background()
	if err := docker.RemoveContainer(ctx, name); err != nil {
		return err
	}

	if update {
		return nil
	}

	// 路径处理
	if path := docker.GetDir(name, "/config"); !file.CheckNotExist(path) {
		return file.RMDir(path)
	}

	return nil
}

// 停止镜像
func (ds *dockerService) StopContainer(id string) error {
	ctx := context.Background()
	return docker.StopContainer(ctx, id)
}

// 启动容器
func (ds *dockerService) StartContainer(name string) error {
	ctx := context.Background()
	return docker.StartContainer(ctx, name)
}

// 查看日志
func (ds *dockerService) GetContainerLog(name string) ([]byte, error) {
	cli, err := client2.NewClientWithOpts(client2.FromEnv, client2.WithAPIVersionNegotiation())
	if err != nil {
		return []byte(""), err
	}
	defer cli.Close()
	// body, err := cli.ContainerAttach(context.Background(), name, types.ContainerAttachOptions{Logs: true, Stream: false, Stdin: false, Stdout: false, Stderr: false})
	body, err := cli.ContainerLogs(context.Background(), name, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return []byte(""), err
	}

	defer body.Close()
	content, err := io.ReadAll(body)
	if err != nil {
		return []byte(""), err
	}
	return content, nil
}

func (ds *dockerService) GetContainerByName(name string) (*types.Container, error) {
	cli, _ := client2.NewClientWithOpts(client2.FromEnv, client2.WithAPIVersionNegotiation())
	defer cli.Close()
	filter := filters.NewArgs()
	filter.Add("name", name)
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true, Filters: filter})
	if err != nil {
		return &types.Container{}, err
	}
	if len(containers) == 0 {
		return &types.Container{}, errors.New("not found")
	}
	return &containers[0], nil
}

// 获取容器详情
func (ds *dockerService) DescribeContainer(ctx context.Context, nameOrID string) (*types.ContainerJSON, error) {
	return docker.Container(ctx, nameOrID)
}

// 更新容器名称
// param name 容器名称
// param id 老的容器名称
func (ds *dockerService) RenameContainer(name, id string) (err error) {
	ctx := context.Background()
	return docker.RenameContainer(ctx, id, name)
}

// 获取网络列表
func (ds *dockerService) GetNetworkList() []types.NetworkResource {
	cli, _ := client2.NewClientWithOpts(client2.FromEnv, client2.WithAPIVersionNegotiation())
	defer cli.Close()
	networks, _ := cli.NetworkList(context.Background(), types.NetworkListOptions{})
	return networks
}

func (ds *dockerService) GetServerInfo() (types.Info, error) {
	cli, err := client2.NewClientWithOpts(client2.FromEnv, client2.WithAPIVersionNegotiation())
	if err != nil {
		return types.Info{}, err
	}
	defer cli.Close()

	return cli.Info(context.Background())
}

func NewDockerService() DockerService {
	return &dockerService{}
}

func getV1AppStoreID(m *types.Container) uint {
	if appStoreIDString, ok := m.Labels[common.ContainerLabelV1AppStoreID]; ok {
		appStoreID, err := strconv.Atoi(appStoreIDString)
		if err != nil {
			logger.Info("failed to convert v1 app store id", zap.Error(err), zap.String("appStoreIDString", appStoreIDString), zap.String("containerID", m.ID), zap.String("containerName", m.Names[0]))
		}

		if appStoreID < 0 {
			appStoreID = 0
		}

		return uint(appStoreID)
	}

	logger.Info("the container does not have a v1 app store id", zap.String("containerID", m.ID), zap.String("containerName", m.Names[0]))
	return 0
}
