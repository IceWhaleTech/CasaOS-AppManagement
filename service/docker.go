package service

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.uber.org/zap"

	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"

	//"github.com/containerd/containerd/oci"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	client2 "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type DockerService interface {
	DockerPullImage(imageName string, icon, name string) error
	IsExistImage(imageName string) bool
	DockerContainerCreate(m model.CustomizationPostData, id string) (containerID string, err error)
	DockerContainerCopyCreate(info *types.ContainerJSON) (containerID string, err error)
	DockerContainerStart(name string) error
	DockerContainerStats(name string) (string, error)
	DockerListByName(name string) (*types.Container, error)
	DockerListByImage(image, version string) (*types.Container, error)
	DockerContainerInfo(name string) (*types.ContainerJSON, error)
	DockerImageRemove(name string) error
	DockerContainerRemove(name string, update bool) error
	DockerContainerStop(id string) error
	DockerContainerUpdateName(name, id string) (err error)
	DockerContainerUpdate(m model.CustomizationPostData, id string) (err error)
	DockerContainerLog(name string) ([]byte, error)
	DockerContainerList() []types.Container
	DockerNetworkModelList() []types.NetworkResource
	GetDockerInfo() (types.Info, error)
}

type dockerService struct{}

func (ds *dockerService) DockerContainerList() []types.Container {
	cli, err := client2.NewClientWithOpts(client2.FromEnv, client2.WithTimeout(time.Second*5))
	if err != nil {
		return nil
	}
	defer cli.Close()
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return containers
	}
	return containers
}

func Exec(container, row, col string) (hr types.HijackedResponse, err error) {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	ctx := context.Background()
	// 执行/bin/bash命令
	ir, err := cli.ContainerExecCreate(ctx, container, types.ExecConfig{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Env:          []string{"COLUMNS=" + col, "LINES=" + row},
		Cmd:          []string{"/bin/bash"},
		Tty:          true,
	})
	if err != nil {
		return
	}
	// 附加到上面创建的/bin/bash进程中
	hr, err = cli.ContainerExecAttach(ctx, ir.ID, types.ExecStartCheck{Detach: false, Tty: true})
	if err != nil {
		return
	}
	return
}

// 正式内容

// 检查镜像是否存在
func (ds *dockerService) IsExistImage(imageName string) bool {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return false
	}
	defer cli.Close()
	filter := filters.NewArgs()
	filter.Add("reference", imageName)

	list, err := cli.ImageList(context.Background(), types.ImageListOptions{Filters: filter})

	if err == nil && len(list) > 0 {
		return true
	}

	return false
}

// 安装镜像
func (ds *dockerService) DockerPullImage(imageName string, icon, name string) error {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return err
	}
	defer cli.Close()
	out, err := cli.ImagePull(context.Background(), imageName, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer out.Close()
	if err != nil {
		return err
	}
	// io.Copy()
	buf := make([]byte, 2048*4)
	for {
		_, err := out.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Println("read error:", err)
			}
			break
		}
		if len(icon) > 0 && len(name) > 0 {
			// notify := notify.Application{}
			// notify.Icon = icon
			// notify.Name = name
			// notify.State = "PULLING"
			// notify.Type = "INSTALL"
			// notify.Finished = false
			// notify.Success = true
			// notify.Message = string(buf[:n])
			// TODO - MyService.Notify().SendInstallAppBySocket(notify)
		}

	}
	return err
}

func (ds *dockerService) DockerContainerCopyCreate(info *types.ContainerJSON) (containerID string, err error) {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return "", err
	}
	defer cli.Close()

	config := &network.NetworkingConfig{EndpointsConfig: info.NetworkSettings.Networks}

	container, err := cli.ContainerCreate(context.Background(), info.Config, info.HostConfig, config, nil, info.Name)
	if err != nil {
		return "", err
	}
	return container.ID, err
}

// param imageName 镜像名称
// param containerDbId 数据库的id
// param port 容器内部主端口
// param mapPort 容器主端口映射到外部的端口
// param tcp 容器其他tcp端口
// param udp 容器其他udp端口
func (ds *dockerService) DockerContainerCreate(m model.CustomizationPostData, id string) (containerID string, err error) {
	if len(m.NetworkModel) == 0 {
		m.NetworkModel = "bridge"
	}

	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return "", err
	}

	defer cli.Close()
	ports := make(nat.PortSet)
	portMaps := make(nat.PortMap)

	for _, portMap := range m.Ports {
		if !lo.Contains([]string{"tcp", "udp", "both"}, portMap.Protocol) {
			message := "unknown protocol"
			logger.Error(message, zap.String("protocol", portMap.Protocol))
			return "", errors.New(message)
		}

		protocols := strings.Replace(portMap.Protocol, "both", "tcp,udp", -1)
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
			// TODO - envArr = append(envArr, e.Name+"="+env_helper.ReplaceDefaultENV(e.Value, MyService.System().GetTimeZone()))
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

// 删除容器
func (ds *dockerService) DockerContainerRemove(name string, update bool) error {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return err
	}
	defer cli.Close()
	err = cli.ContainerRemove(context.Background(), name, types.ContainerRemoveOptions{})

	// 路径处理
	if !update {
		path := docker.GetDir(name, "/config")
		if !file.CheckNotExist(path) {
			if err := file.RMDir(path); err != nil {
				return err
			}
		}
	}

	if err != nil {
		return err
	}

	return err
}

// 删除镜像
func (ds *dockerService) DockerImageRemove(name string) error {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return err
	}
	defer cli.Close()
	imageList, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		return err
	}

	imageID := ""

Loop:
	for _, ig := range imageList {
		for _, i := range ig.RepoTags {
			if i == name {
				imageID = ig.ID
				break Loop
			}
		}
	}
	_, err = cli.ImageRemove(context.Background(), imageID, types.ImageRemoveOptions{})
	return err
}

func DockerImageRemove(name string) error {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return err
	}
	defer cli.Close()

	imageList, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		return err
	}

	imageID := ""

Loop:
	for _, ig := range imageList {
		fmt.Println(ig.RepoDigests)
		fmt.Println(ig.Containers)
		for _, i := range ig.RepoTags {
			if i == name {
				imageID = ig.ID
				break Loop
			}
		}
	}
	_, err = cli.ImageRemove(context.Background(), imageID, types.ImageRemoveOptions{})
	return err
}

// 停止镜像
func (ds *dockerService) DockerContainerStop(id string) error {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return err
	}
	defer cli.Close()
	err = cli.ContainerStop(context.Background(), id, nil)
	return err
}

// 启动容器
func (ds *dockerService) DockerContainerStart(name string) error {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return err
	}
	defer cli.Close()
	err = cli.ContainerStart(context.Background(), name, types.ContainerStartOptions{})
	return err
}

// 查看日志
func (ds *dockerService) DockerContainerLog(name string) ([]byte, error) {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
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
	content, err := ioutil.ReadAll(body)
	// content, err := ioutil.ReadAll(body)
	if err != nil {
		return []byte(""), err
	}
	return content, nil
}

func DockerContainerStats1() error {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return err
	}
	defer cli.Close()
	dss, err := cli.ContainerStats(context.Background(), "dockermysql", false)
	if err != nil {
		return err
	}
	defer dss.Body.Close()
	sts, err := ioutil.ReadAll(dss.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(sts))
	return nil
}

// 获取容器状态
func (ds *dockerService) DockerContainerStats(name string) (string, error) {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return "", err
	}
	defer cli.Close()
	dss, err := cli.ContainerStats(context.Background(), name, false)
	if err != nil {
		return "", err
	}
	defer dss.Body.Close()
	sts, err := ioutil.ReadAll(dss.Body)
	if err != nil {
		return "", err
	}
	return string(sts), nil
}

func (ds *dockerService) DockerListByName(name string) (*types.Container, error) {
	cli, _ := client2.NewClientWithOpts(client2.FromEnv)
	defer cli.Close()
	filter := filters.NewArgs()
	filter.Add("name", name)
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{Filters: filter})
	if err != nil {
		return &types.Container{}, err
	}
	if len(containers) == 0 {
		return &types.Container{}, errors.New("not found")
	}
	return &containers[0], nil
}

func (ds *dockerService) DockerListByImage(image, version string) (*types.Container, error) {
	cli, _ := client2.NewClientWithOpts(client2.FromEnv)
	defer cli.Close()
	filter := filters.NewArgs()
	filter.Add("ancestor", image+":"+version)
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{Filters: filter})
	if err != nil {
		return nil, err
	}
	if len(containers) == 0 {
		return nil, nil
	}
	return &containers[0], nil
}

// 获取容器详情
func (ds *dockerService) DockerContainerInfo(name string) (*types.ContainerJSON, error) {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return &types.ContainerJSON{}, err
	}
	defer cli.Close()
	d, err := cli.ContainerInspect(context.Background(), name)
	if err != nil {
		return &types.ContainerJSON{}, err
	}
	return &d, nil
}

// 更新容器
// param shares cpu优先级
// param containerDbId 数据库的id
// param port 容器内部主端口
// param mapPort 容器主端口映射到外部的端口
// param tcp 容器其他tcp端口
// param udp 容器其他udp端口
func (ds *dockerService) DockerContainerUpdate(m model.CustomizationPostData, id string) (err error) {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return err
	}
	defer cli.Close()
	// 重启策略
	rp := container.RestartPolicy{
		Name:              "",
		MaximumRetryCount: 0,
	}
	if len(m.Restart) > 0 {
		rp.Name = m.Restart
	}
	res := container.Resources{}

	if m.Memory > 0 {
		res.Memory = m.Memory * 1024 * 1024
		res.MemorySwap = -1
	}
	if m.CPUShares > 0 {
		res.CPUShares = m.CPUShares
	}
	for _, p := range m.Devices {
		res.Devices = append(res.Devices, container.DeviceMapping{PathOnHost: p.Path, PathInContainer: p.ContainerPath, CgroupPermissions: "rwm"})
	}
	_, err = cli.ContainerUpdate(context.Background(), id, container.UpdateConfig{RestartPolicy: rp, Resources: res})
	if err != nil {
		return err
	}

	return
}

// 更新容器名称
// param name 容器名称
// param id 老的容器名称
func (ds *dockerService) DockerContainerUpdateName(name, id string) (err error) {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return err
	}
	defer cli.Close()

	err = cli.ContainerRename(context.Background(), id, name)
	if err != nil {
		return err
	}
	return
}

// 获取网络列表
func (ds *dockerService) DockerNetworkModelList() []types.NetworkResource {
	cli, _ := client2.NewClientWithOpts(client2.FromEnv)
	defer cli.Close()
	networks, _ := cli.NetworkList(context.Background(), types.NetworkListOptions{})
	return networks
}

func NewDockerService() DockerService {
	return &dockerService{}
}

func (ds *dockerService) GetDockerInfo() (types.Info, error) {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return types.Info{}, err
	}
	defer cli.Close()

	return cli.Info(context.Background())
}

//   ---------------------------------------test------------------------------------
//func ServiceCreate() {
//	cli, err := client2.NewClientWithOpts(client2.FromEnv)
//	r, err := cli.ServiceCreate(context.Background(), swarm.ServiceSpec{}, types.ServiceCreateOptions{})
//	if err != nil {
//		fmt.Println("error", err)
//	}
//
//
//}

// func Containerd() {
// 	// create a new client connected to the default socket path for containerd
// 	cli, err := containerd.New("/run/containerd/containerd.sock")
// 	if err != nil {
// 		fmt.Println("111")
// 		fmt.Println(err)
// 	}
// 	defer cli.Close()

// 	// create a new context with an "example" namespace
// 	ctx := namespaces.WithNamespace(context.Background(), "default")

// 	// pull the redis image from DockerHub
// 	image, err := cli.Pull(ctx, "docker.io/library/busybox:latest", containerd.WithPullUnpack)
// 	if err != nil {
// 		fmt.Println("222")
// 		fmt.Println(err)
// 	}

// 	// create a container
// 	container, err := cli.NewContainer(
// 		ctx,
// 		"test1",
// 		containerd.WithImage(image),
// 		containerd.WithNewSnapshot("redis-server-snapshot1", image),
// 		containerd.WithNewSpec(oci.WithImageConfig(image)),
// 	)

// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	defer container.Delete(ctx, containerd.WithSnapshotCleanup)

// 	// create a task from the container
// 	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	defer task.Delete(ctx)

// 	// make sure we wait before calling start
// 	exitStatusC, err := task.Wait(ctx)
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	// call start on the task to execute the redis server
// 	if err = task.Start(ctx); err != nil {
// 		fmt.Println(err)
// 	}

// 	fmt.Println("执行完成等待")
// 	// sleep for a lil bit to see the logs
// 	time.Sleep(3 * time.Second)

// 	// kill the process and get the exit status
// 	if err = task.Kill(ctx, syscall.SIGTERM); err != nil {
// 		fmt.Println(err)
// 	}

// 	// wait for the process to fully exit and print out the exit status

// 	status := <-exitStatusC
// 	code, _, err := status.Result()
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	fmt.Printf("redis-server exited with status: %d\n", code)

// }
