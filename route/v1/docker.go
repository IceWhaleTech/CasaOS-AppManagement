package v1

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	modelCommon "github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/model/notify"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/IceWhaleTech/CasaOS-Common/utils/port"
	"github.com/IceWhaleTech/CasaOS-Common/utils/ssh"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/samber/lo"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	CheckOrigin:      func(r *http.Request) bool { return true },
	HandshakeTimeout: time.Duration(time.Second * 5),
}

func backgroundProcess(imageName string, m *model.CustomizationPostData) {
	// step：下载镜像
	err := service.MyService.Docker().DockerPullImage(imageName, m.Icon, m.Label)
	if err != nil {
		app := notify.Application{
			Icon:     m.Icon,
			Name:     m.Label,
			State:    "PULLING",
			Type:     "INSTALL",
			Success:  false,
			Finished: false,
			Message:  err.Error(),
		}

		if err := service.MyService.Notify().SendInstallAppBySocket(app); err != nil {
			logger.Error("send install app notify error", zap.Error(err), zap.Any("app", app))
		}
		return
	}

	for !service.MyService.Docker().IsExistImage(m.Image) {
		time.Sleep(time.Second)
	}

	_, err = service.MyService.Docker().DockerContainerCreate(*m, "")
	if err != nil {
		app := notify.Application{
			Icon:     m.Icon,
			Name:     m.Label,
			State:    "STARTING",
			Type:     "INSTALL",
			Success:  false,
			Finished: false,
			Message:  err.Error(),
		}
		if err := service.MyService.Notify().SendInstallAppBySocket(app); err != nil {
			logger.Error("send install app notify error", zap.Error(err), zap.Any("app", app))
		}
		return
	}

	app := notify.Application{
		Icon:     m.Icon,
		Name:     m.Label,
		State:    "STARTING",
		Type:     "INSTALL",
		Success:  true,
		Finished: false,
	}

	if err := service.MyService.Notify().SendInstallAppBySocket(app); err != nil {
		logger.Error("send install app notify error", zap.Error(err), zap.Any("app", app))
	}
	//		echo -e "hellow\nworld" >>

	// step：启动容器
	err = service.MyService.Docker().DockerContainerStart(m.ContainerName)
	if err != nil {
		app := notify.Application{}
		app.Icon = m.Icon
		app.Name = m.Label
		app.State = "STARTING"
		app.Type = "INSTALL"
		app.Success = false
		app.Finished = false
		app.Message = err.Error()

		if err := service.MyService.Notify().SendInstallAppBySocket(app); err != nil {
			logger.Error("send install app notify error", zap.Error(err), zap.Any("app", app))
		}
		return
	}

	// if m.Origin != CUSTOM {
	// 	installLog.Message = "setting upnp"
	// } else {
	// 	installLog.Message = "nearing completion"
	// }
	// service.MyService.Notify().UpdateLog(installLog)

	// step: 启动成功     检查容器状态确认启动成功
	container, err := service.MyService.Docker().DockerContainerInfo(m.ContainerName)
	if err != nil && container.ContainerJSONBase.State.Running {
		app := notify.Application{
			Icon:     m.Icon,
			Name:     m.Label,
			State:    "INSTALLED",
			Type:     "INSTALL",
			Success:  false,
			Finished: true,
			Message:  err.Error(),
		}

		if err := service.MyService.Notify().SendInstallAppBySocket(app); err != nil {
			logger.Error("send install app notify error", zap.Error(err), zap.Any("app", app))
		}
		return
	}

	app = notify.Application{
		Icon:     m.Icon,
		Name:     m.Label,
		State:    "INSTALLED",
		Type:     "INSTALL",
		Success:  true,
		Finished: true,
	}

	if err := service.MyService.Notify().SendInstallAppBySocket(app); err != nil {
		logger.Error("send install app notify error", zap.Error(err), zap.Any("app", app))
	}

	// if m.Origin != "custom" {
	// 	for i := 0; i < len(m.Volumes); i++ {
	// 		m.Volumes[i].Path = docker.GetDir(id, m.Volumes[i].Path)
	// 	}
	// }
	// service.MyService.App().SaveContainer(md)
	config.CasaOSGlobalVariables.AppChange = true
}

// 打开docker的terminal
func DockerTerminal(c *gin.Context) {
	col := c.DefaultQuery("cols", "100")
	row := c.DefaultQuery("rows", "30")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}
	defer conn.Close()
	container := c.Param("id")
	hr, err := service.Exec(container, row, col)
	if err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
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
func InstallApp(c *gin.Context) {
	m := model.CustomizationPostData{}
	if err := c.ShouldBind(&m); err != nil {
		c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: err.Error()})
		return
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
			if _, err := service.MyService.Docker().DockerListByName(m.ContainerName); err != nil {
				break
			}
		}
	} else {
		if _, err := service.MyService.Docker().DockerListByName(m.ContainerName); err == nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.ERROR_APP_NAME_EXIST, Message: common_err.GetMsg(common_err.ERROR_APP_NAME_EXIST)})
			return
		}
	}

	// check port
	if len(m.PortMap) > 0 && m.PortMap != "0" {
		portMap, err := strconv.Atoi(m.PortMap)
		if err != nil {
			c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: err.Error()})
			return
		}

		if !port.IsPortAvailable(portMap, "tcp") {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "Duplicate port:" + m.PortMap})
			return
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

		if !lo.Contains([]string{"tcp", "udp", "both"}, u.Protocol) {
			c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: "Protocol must be tcp or udp or both"})
			return
		}

		protocols := strings.Replace(u.Protocol, "both", "tcp,udp", -1)
		for _, p := range strings.Split(protocols, ",") {
			t, err := strconv.Atoi(u.CommendPort)
			if err != nil {
				logger.Info("host port is not number - will pick port number randomly", zap.String("port", u.CommendPort))
			}

			if !port.IsPortAvailable(t, p) {
				c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "Duplicate port:" + u.CommendPort})
				return
			}
		}
	}

	if m.Origin == CUSTOM {
		for _, device := range m.Devices {
			if file.CheckNotExist(device.Path) {
				c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.DEVICE_NOT_EXIST, Message: device.Path + "," + common_err.GetMsg(common_err.DEVICE_NOT_EXIST)})
				return
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
	go backgroundProcess(dockerImage+":"+dockerImageVersion, &m)

	c.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: m.Label})
}

// @Summary 卸载app
// @Produce  application/json
// @Accept multipart/form-data
// @Tags app
// @Param  id path string true "容器id"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/uninstall/{id} [delete]
func UnInstallApp(c *gin.Context) {
	appID := c.Param("id")

	if len(appID) == 0 {
		c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
		return
	}
	// info := service.MyService.App().GetUninstallInfo(appId)

	info, err := service.MyService.Docker().DockerContainerInfo(appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	// step：停止容器
	err = service.MyService.Docker().DockerContainerStop(appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.UNINSTALL_APP_ERROR, Message: common_err.GetMsg(common_err.UNINSTALL_APP_ERROR), Data: err.Error()})
		return
	}

	err = service.MyService.Docker().DockerContainerRemove(appID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.UNINSTALL_APP_ERROR, Message: common_err.GetMsg(common_err.UNINSTALL_APP_ERROR), Data: err.Error()})
		return
	}

	// step：remove image
	if err := service.MyService.Docker().DockerImageRemove(info.Config.Image); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.UNINSTALL_APP_ERROR, Message: common_err.GetMsg(common_err.UNINSTALL_APP_ERROR), Data: err.Error()})
		return
	}

	if info.Config.Labels["origin"] != "custom" {
		// step: 删除文件夹
		for _, v := range info.Mounts {
			if strings.Contains(v.Source, info.Name) {
				path := filepath.Join(strings.Split(v.Source, info.Name)[0], info.Name)
				if err := file.RMDir(path); err != nil {
					c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.UNINSTALL_APP_ERROR, Message: err.Error()})
					return
				}
			}
		}
	}
	config.CasaOSGlobalVariables.AppChange = true

	notify := notify.Application{
		Icon:     info.Config.Labels["icon"],
		Name:     strings.ReplaceAll(info.Name, "/", ""),
		State:    "FINISHED",
		Type:     "UNINSTALL",
		Success:  true,
		Finished: true,
	}

	if err := service.MyService.Notify().SendUninstallAppBySocket(notify); err != nil {
		logger.Error("send uninstall app notify error", zap.Error(err), zap.Any("notify", notify))
	}

	c.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
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
func ChangAppState(c *gin.Context) {
	appID := c.Param("id")
	if len(appID) == 0 {
		c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: "id should not be empty"})
		return
	}

	js := make(map[string]string)
	if err := c.ShouldBind(&js); err != nil {
		c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: err.Error()})
		return
	}

	state, ok := js["state"]
	if !ok || len(state) == 0 {
		c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: "`state` should exist and it should not be empty"})
		return
	}

	switch state {

	case "start":
		if err := service.MyService.Docker().DockerContainerStart(appID); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}
	case "restart":
		if err := service.MyService.Docker().DockerContainerStop(appID); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

		if err := service.MyService.Docker().DockerContainerStart(appID); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

	case "stop":
		if err := service.MyService.Docker().DockerContainerStop(appID); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: "`state` should be start, stop or restart"})
	}

	info, err := service.MyService.App().GetContainerInfo(appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: info.State})
}

// @Summary 查看容器日志
// @Produce  application/json
// @Accept application/json
// @Tags app
// @Param  id path string true "appid"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/logs/{id} [get]
func ContainerLog(c *gin.Context) {
	appID := c.Param("id")

	log, err := service.MyService.Docker().DockerContainerLog(appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: string(log)})
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
func GetContainerState(c *gin.Context) {
	id := c.Param("id")
	// t := c.DefaultQuery("type", "0")
	containerInfo, e := service.MyService.App().GetSimpleContainerInfo(id)
	if e != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: e.Error()})
		return
	}

	data := make(map[string]interface{})

	data["state"] = containerInfo.State

	c.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
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
func UpdateSetting(c *gin.Context) {
	id := c.Param("id")
	if len(id) == 0 {
		c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: "id should not be empty"})
		return
	}

	m := model.CustomizationPostData{}
	if err := c.ShouldBind(&m); err != nil {
		c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: err.Error()})
		return
	}

	portMap, err := strconv.Atoi(m.PortMap)
	if err != nil {
		c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: err.Error()})
		return
	}

	if err := service.MyService.Docker().DockerContainerStop(id); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	if !port.IsPortAvailable(portMap, "tcp") {
		if err := service.MyService.Docker().DockerContainerStart(id); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "Duplicate port:" + m.PortMap})
		return
	}

	for _, u := range m.Ports {
		t, err := strconv.Atoi(u.CommendPort)
		if err != nil {
			logger.Info("host port is not number - will pick port number randomly", zap.String("port", u.CommendPort))
		}

		if !lo.Contains([]string{"tcp", "udp", "both"}, u.Protocol) {
			c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: "protocol should be tcp, udp or both"})
			return
		}

		protocols := strings.Replace(u.Protocol, "both", "tcp,udp", -1)

		for _, p := range strings.Split(protocols, ",") {
			if !port.IsPortAvailable(t, p) {
				if err := service.MyService.Docker().DockerContainerStart(id); err != nil {
					c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
					return
				}

				c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "Duplicate port:" + u.CommendPort})
				return
			}
		}
	}

	if err := service.MyService.Docker().DockerContainerUpdateName(id, id); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	containerID, err := service.MyService.Docker().DockerContainerCreate(m, id)
	if err != nil {
		if err := service.MyService.Docker().DockerContainerUpdateName(m.ContainerName, id); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

		if err := service.MyService.Docker().DockerContainerStart(id); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	// step：启动容器
	if err = service.MyService.Docker().DockerContainerStart(containerID); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	if err := service.MyService.Docker().DockerContainerRemove(id, true); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

// @Summary update app version
// @Produce  application/json
// @Accept multipart/form-data
// @Tags app
// @Param  id path string true "容器id"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/update/{id} [put]
func PutAppUpdate(c *gin.Context) {
	id := c.Param("id")
	if len(id) == 0 {
		c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
		return
	}

	inspect, err := service.MyService.Docker().DockerContainerInfo(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return

	}

	imageLatest := strings.Split(inspect.Config.Image, ":")[0] + ":latest"
	if err := service.MyService.Docker().DockerPullImage(imageLatest, "", ""); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return

	}

	if err := service.MyService.Docker().DockerContainerStop(id); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	if err := service.MyService.Docker().DockerContainerUpdateName(id, id); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	inspect.Image = imageLatest
	inspect.Config.Image = imageLatest

	containerID, err := service.MyService.Docker().DockerContainerCopyCreate(inspect)
	if err != nil {
		if err := service.MyService.Docker().DockerContainerUpdateName(inspect.Name, id); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

		if err := service.MyService.Docker().DockerContainerStart(id); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	// step：启动容器
	if err := service.MyService.Docker().DockerContainerStart(containerID); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	if err := service.MyService.Docker().DockerContainerRemove(id, true); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	delete(service.NewVersionApp, id)

	c.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

func GetDockerNetworks(c *gin.Context) {
	networks := service.MyService.Docker().DockerNetworkModelList()
	list := []map[string]string{}
	for _, network := range networks {
		if network.Driver != "null" {
			list = append(list, map[string]string{"name": network.Name, "driver": network.Driver, "id": network.ID})
		}
	}

	c.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: list})
}

// @Produce  application/json
// @Accept application/json
// @Tags app
// @Param  id path string true "appid"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/update/{id}/info [get]
func ContainerUpdateInfo(c *gin.Context) {
	appID := c.Param("id")
	info, err := service.MyService.Docker().DockerContainerInfo(appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: err.Error()})
		return
	}

	var port model.PortArray

	for k, v := range info.HostConfig.PortBindings {
		temp := model.PortMap{
			CommendPort:   v[0].HostPort,
			ContainerPort: k.Port(),

			Protocol: k.Proto(),
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
		if len(showENVList) > 0 && info.Config.Labels["origin"] != "local" {
			if _, ok := showENVMap[strings.Split(v, "=")[0]]; ok {
				temp := model.Env{
					Name:  strings.Split(v, "=")[0],
					Value: strings.Split(v, "=")[1],
				}
				envs = append(envs, temp)
			}
		} else {
			temp := model.Env{
				Name:  strings.Split(v, "=")[0],
				Value: strings.Split(v, "=")[1],
			}
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

	name := info.Config.Labels["name"]
	if len(name) == 0 {
		name = strings.ReplaceAll(info.Name, "/", "")
	}

	m := model.CustomizationPostData{
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
		Icon:          info.Config.Labels["icon"],
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

	c.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: m})
}
