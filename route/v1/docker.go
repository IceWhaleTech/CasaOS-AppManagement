package v1

import (
	"context"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	v2 "github.com/IceWhaleTech/CasaOS-AppManagement/route/v2"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	v1 "github.com/IceWhaleTech/CasaOS-AppManagement/service/v1"
	modelCommon "github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-Common/utils/port"
	"github.com/IceWhaleTech/CasaOS-Common/utils/ssh"
	"github.com/IceWhaleTech/CasaOS-Common/utils/systemctl"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/errdefs"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
)

const (
	dockerRootDirFilePath             = "/var/lib/casaos/docker_root"
	dockerDaemonConfigurationFilePath = "/etc/docker/daemon.json"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var upgrader = websocket.Upgrader{
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	CheckOrigin:      func(r *http.Request) bool { return true },
	HandshakeTimeout: time.Duration(time.Second * 5),
}

// 打开docker的terminal
func DockerTerminal(ctx echo.Context) error {
	col := v2.DefaultQuery(ctx, "cols", "100")
	row := v2.DefaultQuery(ctx, "rows", "30")
	conn, err := upgrader.Upgrade(ctx.Response().Writer, ctx.Request(), nil)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}
	defer conn.Close()
	container := ctx.Param("id")
	hr, err := service.MyService.Docker().CreateContainerShellSession(container, row, col)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}
	// 关闭I/O流
	defer hr.Close()
	// 退出进程
	defer func() {
		if _, err := hr.Conn.Write([]byte("exit\r")); err != nil {
			logger.Error("error when trying `exit` to container", zap.Error(err))
		}
	}()
	go func() {
		ssh.WsWriterCopy(hr.Conn, conn)
	}()
	ssh.WsReaderCopy(conn, hr.Conn)
	return nil
}

// @Summary 安装app(该接口需要post json数据)
// @Produce  application/json
// @Accept application/json
// @Tags app
// @Param  id path int true "id"
// @Param  port formData int true "主端口"
// @Param  tcp formData string false "tcp端口"
// @Param  udp formData string false "udp端口"
// @Param  env formData string false "环境变量"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/install [post]
func InstallApp(ctx echo.Context) error {
	m := model.CustomizationPostData{}
	if err := ctx.Bind(&m); err != nil {
		return ctx.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: err.Error()})
	}

	const CUSTOM = "custom"
	var dockerImage string
	var dockerImageVersion string

	// check app name is exist
	if len(m.Protocol) == 0 {
		m.Protocol = "http"
	}
	m.ContainerName = strings.Replace(m.Label, " ", "_", -1)
	if m.Origin != CUSTOM {
		oldName := m.ContainerName
		oldLabel := m.Label
		for i := 0; true; i++ {
			if i != 0 {
				m.ContainerName = oldName + "-" + strconv.Itoa(i)
				m.Label = oldLabel + "-" + strconv.Itoa(i)
			}
			if _, err := service.MyService.Docker().GetContainerByName(m.ContainerName); err != nil {
				break
			}
		}
	} else {
		if _, err := service.MyService.Docker().GetContainerByName(m.ContainerName); err == nil {
			return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.ERROR_APP_NAME_EXIST, Message: common_err.GetMsg(common_err.ERROR_APP_NAME_EXIST)})
		}
	}

	// check port
	if len(m.PortMap) > 0 && m.PortMap != "0" {
		portMap, err := strconv.Atoi(m.PortMap)
		if err != nil {
			return ctx.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: err.Error()})
		}

		if !port.IsPortAvailable(portMap, "tcp") {
			return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "Duplicate port:" + m.PortMap})
		}
	}

	imageArr := strings.Split(m.Image, ":")
	if len(imageArr) == 2 {
		dockerImage = imageArr[0]
		dockerImageVersion = imageArr[1]
	} else {
		dockerImage = m.Image
		dockerImageVersion = "latest"
	}
	m.Image = dockerImage + ":" + dockerImageVersion
	for _, u := range m.Ports {
		protocol := strings.ToLower(u.Protocol)

		if !lo.Contains([]string{"tcp", "udp", "both"}, protocol) {
			return ctx.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: "Protocol must be tcp or udp or both"})
		}

		protocols := strings.Replace(protocol, "both", "tcp,udp", -1)
		for _, p := range strings.Split(protocols, ",") {
			t, err := strconv.Atoi(u.CommendPort)
			if err != nil {
				logger.Info("host port is not number - will pick port number randomly", zap.String("port", u.CommendPort))
			}

			if !port.IsPortAvailable(t, p) {
				return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "Duplicate port:" + u.CommendPort})
			}
		}
	}

	if m.Origin == CUSTOM {
		for _, device := range m.Devices {
			if file.CheckNotExist(device.Path) {
				return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.DEVICE_NOT_EXIST, Message: device.Path + "," + common_err.GetMsg(common_err.DEVICE_NOT_EXIST)})
			}
		}
	} else {
		dev := []model.PathMap{}
		for _, device := range dev {
			if !file.CheckNotExist(device.Path) {
				dev = append(dev, device)
			}
		}
		m.Devices = dev
	}

	id := uuid.NewV4().String()
	m.CustomID = id

	imageName := dockerImage + ":" + dockerImageVersion

	httpCtx := common.WithProperties(context.Background(), v2.PropertiesFromQueryParams(ctx))

	eventProperties := common.PropertiesFromContext(httpCtx)
	eventProperties[common.PropertyTypeAppName.Name] = m.Label
	eventProperties[common.PropertyTypeAppIcon.Name] = m.Icon
	eventProperties[common.PropertyTypeImageName.Name] = imageName

	go func() {
		go service.PublishEventWrapper(httpCtx, common.EventTypeAppInstallBegin, nil)

		defer service.PublishEventWrapper(httpCtx, common.EventTypeAppInstallEnd, nil)

		if err := pullAndInstall(httpCtx, imageName, &m); err != nil {
			go service.PublishEventWrapper(httpCtx, common.EventTypeAppInstallError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})
		}
	}()

	return ctx.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: m.Label})
}

// @Summary 卸载app
// @Produce  application/json
// @Accept multipart/form-data
// @Tags app
// @Param  id path string true "容器id"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/uninstall/{id} [delete]
func UninstallApp(ctx echo.Context) error {
	containerID := ctx.Param("id")
	if len(containerID) == 0 {
		return ctx.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
	}

	j := make(map[string]bool)
	if err := (&echo.DefaultBinder{}).BindBody(ctx, &j); err != nil {
		return ctx.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: err.Error()})
	}

	isDelete, ok := j["delete_config_folder"]
	if !ok {
		isDelete = false
	}

	httpCtx := common.WithProperties(context.Background(), v2.PropertiesFromQueryParams(ctx))

	container, err := service.MyService.Docker().DescribeContainer(httpCtx, containerID)
	if err != nil {
		if _, ok := err.(errdefs.ErrNotFound); ok {
			return ctx.JSON(http.StatusNotFound, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		}

		return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	eventProperties := common.PropertiesFromContext(httpCtx)
	eventProperties[common.PropertyTypeAppName.Name] = v1.AppName(container)
	eventProperties[common.PropertyTypeAppIcon.Name] = v1.AppIcon(container)

	go func() {
		go service.PublishEventWrapper(httpCtx, common.EventTypeAppUninstallBegin, nil)

		defer service.PublishEventWrapper(httpCtx, common.EventTypeAppUninstallEnd, nil)

		if err := uninstall(httpCtx, container, isDelete); err != nil {
			go service.PublishEventWrapper(httpCtx, common.EventTypeAppUninstallError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})
		}
	}()

	return ctx.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

// @Summary 修改app状态
// @Produce  application/json
// @Accept multipart/form-data
// @Tags app
// @Param  id path string true "appid"
// @Param  state query string false "是否停止 start stop restart"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/state/{id} [put]
func ChangAppState(ctx echo.Context) error {
	appID := ctx.Param("id")
	if len(appID) == 0 {
		return ctx.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: "id should not be empty"})
	}

	js := make(map[string]string)
	if err := ctx.Bind(&js); err != nil {
		return ctx.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: err.Error()})
	}

	state, ok := js["state"]
	if !ok || len(state) == 0 {
		return ctx.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: "`state` should exist and it should not be empty"})
	}

	switch state {

	case "start":
		if err := service.MyService.Docker().StartContainer(appID); err != nil {
			return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		}
	case "restart":
		if err := service.MyService.Docker().StopContainer(appID); err != nil {
			return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		}

		if err := service.MyService.Docker().StartContainer(appID); err != nil {
			return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		}

	case "stop":
		if err := service.MyService.Docker().StopContainer(appID); err != nil {
			return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		}

	default:
		return ctx.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: "`state` should be start, stop or restart"})
	}

	info, err := service.MyService.Docker().GetContainer(appID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	return ctx.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: info.State})
}

// @Summary 查看容器日志
// @Produce  application/json
// @Accept application/json
// @Tags app
// @Param  id path string true "appid"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/logs/{id} [get]
func ContainerLog(ctx echo.Context) error {
	appID := ctx.Param("id")

	log, err := service.MyService.Docker().GetContainerLog(appID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	return ctx.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: string(log)})
}

// @Summary 获取容器状态
// @Produce  application/json
// @Accept application/json
// @Tags app
// @Param  id path string true "容器id"
// @Param  type query string false "type=1"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/state/{id} [get]
func GetContainerState(ctx echo.Context) error {
	id := ctx.Param("id")
	// t := v2.DefaultQuery(ctx, "type", "0")
	containerInfo, e := service.MyService.Docker().GetContainer(id)
	if e != nil {
		return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: e.Error()})
	}

	data := make(map[string]interface{})

	data["state"] = containerInfo.State

	return ctx.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}

// @Summary 更新设置
// @Produce  application/json
// @Accept multipart/form-data
// @Tags app
// @Param  id path string true "容器id"
// @Param  shares formData string false "cpu权重"
// @Param  mem formData string false "内存大小MB"
// @Param  restart formData string false "重启策略"
// @Param  label formData string false "应用名称"
// @Param  position formData bool true "是否放到首页"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/update/{id}/setting [put]
func UpdateSetting(ctx echo.Context) error {
	id := ctx.Param("id")
	if len(id) == 0 {
		return ctx.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: "id should not be empty"})
	}

	m := model.CustomizationPostData{}
	if err := ctx.Bind(&m); err != nil {
		return ctx.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: err.Error()})
	}

	if err := service.MyService.Docker().StopContainer(id); err != nil {
		return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	for _, u := range m.Ports {
		protocol := strings.ToLower(u.Protocol)

		if !lo.Contains([]string{"tcp", "udp", "both"}, protocol) {
			return ctx.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: "protocol should be tcp, udp or both"})
		}
	}

	if err := service.MyService.Docker().RenameContainer(id, id); err != nil {
		logger.Error("rename container error: ", zap.Error(err))
	}

	containerID, err := service.MyService.Docker().CreateContainer(m, id)
	if err != nil {
		if err := service.MyService.Docker().RenameContainer(m.ContainerName, id); err != nil {
			logger.Error("rename container error: ", zap.Error(err))
		}

		if err := service.MyService.Docker().StartContainer(id); err != nil {
			return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		}

		return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	// step：启动容器
	if err = service.MyService.Docker().StartContainer(containerID); err != nil {
		return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	if err := service.MyService.Docker().RemoveContainer(id, true); err != nil {
		return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	return ctx.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

func GetDockerNetworks(ctx echo.Context) error {
	networks := service.MyService.Docker().GetNetworkList()
	list := []map[string]string{}
	for _, network := range networks {
		if network.Driver != "null" {
			list = append(list, map[string]string{"name": network.Name, "driver": network.Driver, "id": network.ID})
		}
	}

	return ctx.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: list})
}

func ToComposeYAML(ctx echo.Context) error {
	appID := ctx.Param("id")

	httpCtx := common.WithProperties(context.Background(), v2.PropertiesFromQueryParams(ctx))

	info, err := service.MyService.Docker().DescribeContainer(httpCtx, appID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	m := v1.GetCustomizationPostData(*info)

	return YAML(ctx, http.StatusOK, m.Compose())
}

// @Produce  application/json
// @Accept application/json
// @Tags app
// @Param  id path string true "appid"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/update/{id}/info [get]
func ContainerUpdateInfo(ctx echo.Context) error {
	appID := ctx.Param("id")

	httpCtx := common.WithProperties(context.Background(), v2.PropertiesFromQueryParams(ctx))

	info, err := service.MyService.Docker().DescribeContainer(httpCtx, appID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	m := v1.GetCustomizationPostData(*info)

	return ctx.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: m})
}

// @Summary 我的应用列表
// @Produce  application/json
// @Accept application/json
// @Tags app
// @Security ApiKeyAuth
// @Param  index query int false "index"
// @Param  size query int false "size"
// @Param  position query bool false "是否是首页应用"
// @Success 200 {string} string "ok"
// @Router /app/my/list [get]
func MyAppList(ctx echo.Context) error {
	name := ctx.QueryParam("name")
	image := ctx.QueryParam("image")
	state := ctx.QueryParam("state")

	casaOSApps, localApps := service.MyService.Docker().GetContainerAppList(&name, &image, &state)
	data := make(map[string]interface{}, 2)
	data["casaos_apps"] = casaOSApps
	data["local_apps"] = localApps

	return ctx.JSON(common_err.SUCCESS, &modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}

// NOTE: the API is a temporary and internal API. It will be deleted in the future.
// the API is for archive v1 app for rebuilt v2 app.
func ArchiveContainer(ctx echo.Context) error {
	appID := ctx.Param("id")

	if err := service.MyService.Docker().StopContainer(appID); err != nil {
		return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	// get container name
	container, err := service.MyService.Docker().GetContainer(appID)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	if err := service.MyService.Docker().RenameContainer(container.Names[0]+"_old", appID); err != nil {
		return ctx.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
	}

	return ctx.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

// @Summary my app hardware usage list
// @Produce  application/json
// @Accept application/json
// @Tags app
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/usage [get]
func AppUsageList(ctx echo.Context) error {
	list := service.MyService.Docker().GetContainerStats()
	return ctx.JSON(common_err.SUCCESS, &modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: list})
	// return ctx.JSON(common_err.SUCCESS, &modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: nil})
}

func GetDockerDaemonConfiguration(ctx echo.Context) error {
	// info, err := service.MyService.Docker().GetServerInfo()
	// if err != nil {
	// 	return ctx.JSON(common_err.SERVICE_ERROR, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR), Data: err.Error()})
	// 	return
	// }
	data := make(map[string]interface{})

	if file.Exists(dockerRootDirFilePath) {
		buf := file.ReadFullFile(dockerRootDirFilePath)
		err := json.Unmarshal(buf, &data)
		if err != nil {
			return ctx.JSON(common_err.CLIENT_ERROR, &modelCommon.Result{Success: common_err.CLIENT_ERROR, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: err})
		}
	}
	return ctx.JSON(common_err.SUCCESS, &modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}

func PutDockerDaemonConfiguration(ctx echo.Context) error {
	request := make(map[string]interface{})
	if err := ctx.Bind(&request); err != nil {
		return ctx.JSON(http.StatusBadRequest, &modelCommon.Result{Success: common_err.CLIENT_ERROR, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: err})
	}

	value, ok := request["docker_root_dir"]
	if !ok {
		return ctx.JSON(http.StatusBadRequest, &modelCommon.Result{Success: common_err.CLIENT_ERROR, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: "`docker_root_dir` should not empty"})
	}

	dockerConfig := model.DockerDaemonConfigurationModel{}
	if file.Exists(dockerDaemonConfigurationFilePath) {
		byteResult := file.ReadFullFile(dockerDaemonConfigurationFilePath)
		err := json.Unmarshal(byteResult, &dockerConfig)
		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "error when trying to deserialize " + dockerDaemonConfigurationFilePath, Data: err})
		}
	}

	dockerRootDir := value.(string)
	if dockerRootDir == "/" {
		dockerConfig.Root = "" // omitempty - empty string will not be serialized
	} else {
		if !file.Exists(dockerRootDir) {
			return ctx.JSON(http.StatusBadRequest, &modelCommon.Result{Success: common_err.CLIENT_ERROR, Message: common_err.GetMsg(common_err.DIR_NOT_EXISTS), Data: common_err.GetMsg(common_err.DIR_NOT_EXISTS)})
		}

		dockerConfig.Root = filepath.Join(dockerRootDir, "docker")

		if err := file.IsNotExistMkDir(dockerConfig.Root); err != nil {
			return ctx.JSON(http.StatusInternalServerError, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "error when trying to create " + dockerConfig.Root, Data: err})
		}
	}

	buf, err := json.Marshal(request)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, &modelCommon.Result{Success: common_err.CLIENT_ERROR, Message: "error when trying to serialize docker root json", Data: err})
	}

	if err := file.WriteToFullPath(buf, dockerRootDirFilePath, 0o644); err != nil {
		return ctx.JSON(http.StatusInternalServerError, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "error when trying to write " + dockerRootDirFilePath, Data: err})
	}

	buf, err = json.Marshal(dockerConfig)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, &modelCommon.Result{Success: common_err.CLIENT_ERROR, Message: "error when trying to serialize docker config", Data: dockerConfig})
	}

	if err := file.WriteToFullPath(buf, dockerDaemonConfigurationFilePath, 0o644); err != nil {
		return ctx.JSON(http.StatusInternalServerError, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "error when trying to write to " + dockerDaemonConfigurationFilePath, Data: err})
	}

	if err := systemctl.ReloadDaemon(); err != nil {
		return ctx.JSON(http.StatusInternalServerError, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "error when trying to reload systemd daemon"})
	}

	if err := systemctl.StopService("docker"); err != nil {
		return ctx.JSON(http.StatusInternalServerError, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "error when trying to stop docker service"})
	}

	if err := systemctl.StartService("docker"); err != nil {
		return ctx.JSON(http.StatusInternalServerError, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "error when trying to start docker service"})
	}

	return ctx.JSON(http.StatusOK, &modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: request})
}

func pullAndInstall(ctx context.Context, imageName string, m *model.CustomizationPostData) error {
	// step：下载镜像
	if err := func() error {
		go service.PublishEventWrapper(ctx, common.EventTypeImagePullBegin, nil)

		defer service.PublishEventWrapper(ctx, common.EventTypeImagePullEnd, nil)

		if err := service.MyService.Docker().PullImage(ctx, imageName); err != nil {

			go service.PublishEventWrapper(ctx, common.EventTypeImagePullError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})

			return err
		}

		for !service.MyService.Docker().IsExistImage(m.Image) {
			time.Sleep(time.Second)
		}

		return nil
	}(); err != nil {
		return err
	}

	var containerID string

	if err := func() error {
		go service.PublishEventWrapper(ctx, common.EventTypeContainerCreateBegin, nil)

		defer service.PublishEventWrapper(ctx, common.EventTypeContainerCreateEnd, nil)

		_containerID, err := service.MyService.Docker().CreateContainer(*m, "")
		if err != nil {
			go service.PublishEventWrapper(ctx, common.EventTypeContainerCreateError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})
			return err
		}

		containerID = _containerID

		eventProperties := common.PropertiesFromContext(ctx)
		eventProperties[common.PropertyTypeContainerID.Name] = containerID

		return nil
	}(); err != nil {
		return err
	}

	// step：启动容器
	if err := func() error {
		go service.PublishEventWrapper(ctx, common.EventTypeContainerStartBegin, nil)

		defer service.PublishEventWrapper(ctx, common.EventTypeContainerStartEnd, nil)

		if err := service.MyService.Docker().StartContainer(m.ContainerName); err != nil {

			go service.PublishEventWrapper(ctx, common.EventTypeContainerStartError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})
			return err
		}

		return nil
	}(); err != nil {
		return err
	}

	config.CasaOSGlobalVariables.AppChange = true
	return nil
}

func uninstall(ctx context.Context, container *types.ContainerJSON, isDelete bool) error {
	// step：停止容器
	if err := func() error {
		go service.PublishEventWrapper(ctx, common.EventTypeContainerStopBegin, nil)

		defer service.PublishEventWrapper(ctx, common.EventTypeContainerStopEnd, nil)

		if err := service.MyService.Docker().StopContainer(container.ID); err != nil {

			go service.PublishEventWrapper(ctx, common.EventTypeContainerStopError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})

			return err
		}

		return nil
	}(); err != nil {
		return err
	}

	if err := func() error {
		go service.PublishEventWrapper(ctx, common.EventTypeContainerRemoveBegin, nil)

		defer service.PublishEventWrapper(ctx, common.EventTypeContainerRemoveEnd, nil)

		if err := service.MyService.Docker().RemoveContainer(container.ID, false); err != nil {
			go service.PublishEventWrapper(ctx, common.EventTypeContainerRemoveError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})
			return err
		}

		return nil
	}(); err != nil {
		return err
	}

	if err := func() error {
		go service.PublishEventWrapper(ctx, common.EventTypeImageRemoveBegin, nil)

		defer service.PublishEventWrapper(ctx, common.EventTypeImageRemoveEnd, nil)

		if err := service.MyService.Docker().RemoveImage(container.Config.Image); err != nil {
			logger.Error("error when trying to remove docker image", zap.Error(err), zap.String("image", container.Config.Image))

			go service.PublishEventWrapper(ctx, common.EventTypeImageRemoveError, map[string]string{
				common.PropertyTypeMessage.Name: err.Error(),
			})
		}

		return nil
	}(); err != nil {
		return err
	}

	if container.Config.Labels["origin"] != "custom" && isDelete {
		// step: 删除文件夹
		for _, v := range container.Mounts {
			if strings.Contains(v.Source, container.Name) {
				path := filepath.Join(strings.Split(v.Source, container.Name)[0], container.Name)
				if err := file.RMDir(path); err != nil {
					return err
				}
			}
		}
	}
	config.CasaOSGlobalVariables.AppChange = true

	return nil
}
