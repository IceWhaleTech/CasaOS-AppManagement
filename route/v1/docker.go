package v1

import (
	"context"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen/message_bus"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
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
	"github.com/IceWhaleTech/CasaOS-Common/utils/systemctl"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/samber/lo"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
)

const (
	dockerRootDirFilePath             = "/var/lib/casaos/docker_root"
	dockerDaemonConfigurationFilePath = "/etc/docker/daemon.json"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
	CheckOrigin:      func(r *http.Request) bool { return true },
	HandshakeTimeout: time.Duration(time.Second * 5),
}

func publishEventWrapper(ctx context.Context, eventType message_bus.EventType, properties map[string]string) {
	response, err := service.MyService.MessageBus().PublishEventWithResponse(ctx, common.AppManagementServiceName, eventType.Name, properties)
	if err != nil {
		logger.Error("failed to publish event", zap.Error(err))
	}
	defer response.HTTPResponse.Body.Close()

	if response.StatusCode() != http.StatusOK {
		logger.Error("failed to publish event", zap.String("status code", response.Status()))
	}
}

func pullAndCreate(imageName string, m *model.CustomizationPostData) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// publish app installing event
	publishEventWrapper(ctx, common.EventTypeContainerAppInstalling, map[string]string{
		common.PropertyTypeAppName.Name: imageName,
	})

	// step：下载镜像
	if err := service.MyService.Docker().PullImage(imageName, m.Icon, m.Label); err != nil {
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

		publishEventWrapper(ctx, common.EventTypeContainerAppInstallFailed, map[string]string{
			common.PropertyTypeAppName.Name: imageName,
			common.PropertyTypeMessage.Name: err.Error(),
		})
		return
	}

	for !service.MyService.Docker().IsExistImage(m.Image) {
		time.Sleep(time.Second)
	}

	if _, err := service.MyService.Docker().CreateContainer(*m, ""); err != nil {
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

		publishEventWrapper(ctx, common.EventTypeContainerAppInstallFailed, map[string]string{
			common.PropertyTypeAppName.Name: imageName,
			common.PropertyTypeMessage.Name: err.Error(),
		})
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

	// step：启动容器
	if err := service.MyService.Docker().StartContainer(m.ContainerName); err != nil {
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

		publishEventWrapper(ctx, common.EventTypeContainerAppInstallFailed, map[string]string{
			common.PropertyTypeAppName.Name: imageName,
			common.PropertyTypeMessage.Name: err.Error(),
		})
		return
	}

	// if m.Origin != CUSTOM {
	// 	installLog.Message = "setting upnp"
	// } else {
	// 	installLog.Message = "nearing completion"
	// }
	// service.MyService.Notify().UpdateLog(installLog)

	// step: 启动成功     检查容器状态确认启动成功
	container, err := service.MyService.Docker().DescribeContainer(m.ContainerName)
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

		publishEventWrapper(ctx, common.EventTypeContainerAppInstallFailed, map[string]string{
			common.PropertyTypeAppName.Name: imageName,
			common.PropertyTypeMessage.Name: err.Error(),
		})
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

	publishEventWrapper(ctx, common.EventTypeContainerAppInstalled, map[string]string{
		common.PropertyTypeAppID.Name:   container.ID,
		common.PropertyTypeAppName.Name: imageName,
	})

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
	hr, err := service.MyService.Docker().CreateContainerShellSession(container, row, col)
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
			if _, err := service.MyService.Docker().GetContainerByName(m.ContainerName); err != nil {
				break
			}
		}
	} else {
		if _, err := service.MyService.Docker().GetContainerByName(m.ContainerName); err == nil {
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

	go pullAndCreate(dockerImage+":"+dockerImageVersion, &m)

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

	j := make(map[string]bool)
	if err := c.ShouldBind(&j); err != nil {
		c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: err.Error()})
		return
	}

	isDelete, ok := j["delete_config_folder"]
	if !ok {
		isDelete = false
	}

	info, err := service.MyService.Docker().DescribeContainer(appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	// publish app installing event
	publishEventWrapper(c.Request.Context(), common.EventTypeContainerAppUninstalling, map[string]string{
		common.PropertyTypeAppID.Name:   appID,
		common.PropertyTypeAppName.Name: info.Config.Image,
	})

	// step：停止容器
	err = service.MyService.Docker().StopContainer(appID)
	if err != nil {
		publishEventWrapper(c.Request.Context(), common.EventTypeContainerAppUninstallFailed, map[string]string{
			common.PropertyTypeAppID.Name:   appID,
			common.PropertyTypeAppName.Name: info.Config.Image,
			common.PropertyTypeMessage.Name: err.Error(),
		})

		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.UNINSTALL_APP_ERROR, Message: common_err.GetMsg(common_err.UNINSTALL_APP_ERROR), Data: err.Error()})
		return
	}

	err = service.MyService.Docker().RemoveContainer(appID, false)
	if err != nil {
		publishEventWrapper(c.Request.Context(), common.EventTypeContainerAppUninstallFailed, map[string]string{
			common.PropertyTypeAppID.Name:   appID,
			common.PropertyTypeAppName.Name: info.Config.Image,
			common.PropertyTypeMessage.Name: err.Error(),
		})

		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.UNINSTALL_APP_ERROR, Message: common_err.GetMsg(common_err.UNINSTALL_APP_ERROR), Data: err.Error()})
		return
	}

	// step：remove image

	// if err := service.MyService.Docker().RemoveImage(info.Config.Image); err != nil {
	// 	publishEventWrapper(c.Request.Context(), common.EventTypeContainerAppUninstallFailed, map[string]string{
	// 		common.PropertyTypeAppID.Name:   appID,
	// 		common.PropertyTypeAppName.Name: info.Config.Image,
	// 		common.PropertyTypeMessage.Name: err.Error(),
	// 	})

	// 	c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.UNINSTALL_APP_ERROR, Message: common_err.GetMsg(common_err.UNINSTALL_APP_ERROR), Data: err.Error()})
	// 	return
	// }
	if err := service.MyService.Docker().RemoveImage(info.Config.Image); err != nil {
		logger.Error("error when trying to remove docker image", zap.Error(err), zap.String("image", info.Config.Image))
	}

	if info.Config.Labels["origin"] != "custom" && isDelete {
		// step: 删除文件夹
		for _, v := range info.Mounts {
			if strings.Contains(v.Source, info.Name) {
				path := filepath.Join(strings.Split(v.Source, info.Name)[0], info.Name)
				if err := file.RMDir(path); err != nil {
					publishEventWrapper(c.Request.Context(), common.EventTypeContainerAppUninstallFailed, map[string]string{
						common.PropertyTypeAppID.Name:   appID,
						common.PropertyTypeAppName.Name: info.Config.Image,
						common.PropertyTypeMessage.Name: err.Error(),
					})

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

	publishEventWrapper(c.Request.Context(), common.EventTypeContainerAppUninstalled, map[string]string{
		common.PropertyTypeAppID.Name:   appID,
		common.PropertyTypeAppName.Name: info.Config.Image,
	})

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
		if err := service.MyService.Docker().StartContainer(appID); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}
	case "restart":
		if err := service.MyService.Docker().StopContainer(appID); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

		if err := service.MyService.Docker().StartContainer(appID); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

	case "stop":
		if err := service.MyService.Docker().StopContainer(appID); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: "`state` should be start, stop or restart"})
	}

	info, err := service.MyService.Docker().GetContainer(appID)
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

	log, err := service.MyService.Docker().GetContainerLog(appID)
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
	containerInfo, e := service.MyService.Docker().GetContainer(id)
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

	if err := service.MyService.Docker().StopContainer(id); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	if len(m.PortMap) > 0 && m.PortMap != "0" {
		portMap, err := strconv.Atoi(m.PortMap)
		if err != nil {
			c.JSON(http.StatusBadRequest, modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: err.Error()})
			return
		}

		if !port.IsPortAvailable(portMap, "tcp") {
			if err := service.MyService.Docker().StartContainer(id); err != nil {
				c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
				return
			}

			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "Duplicate port:" + m.PortMap})
			return
		}
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
				if err := service.MyService.Docker().StartContainer(id); err != nil {
					c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
					return
				}

				c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "Duplicate port:" + u.CommendPort})
				return
			}
		}
	}

	if err := service.MyService.Docker().RenameContainer(id, id); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	containerID, err := service.MyService.Docker().CreateContainer(m, id)
	if err != nil {
		if err := service.MyService.Docker().RenameContainer(m.ContainerName, id); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

		if err := service.MyService.Docker().StartContainer(id); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	// step：启动容器
	if err = service.MyService.Docker().StartContainer(containerID); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	if err := service.MyService.Docker().RemoveContainer(id, true); err != nil {
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

	inspect, err := service.MyService.Docker().DescribeContainer(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return

	}

	imageLatest := strings.Split(inspect.Config.Image, ":")[0] + ":latest"
	if err := service.MyService.Docker().PullImage(imageLatest, "", ""); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return

	}

	if err := service.MyService.Docker().StopContainer(id); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	if err := service.MyService.Docker().RenameContainer(id, id); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	inspect.Image = imageLatest
	inspect.Config.Image = imageLatest

	containerID, err := service.MyService.Docker().CloneContainer(inspect)
	if err != nil {
		if err := service.MyService.Docker().RenameContainer(inspect.Name, id); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

		if err := service.MyService.Docker().StartContainer(id); err != nil {
			c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	// step：启动容器
	if err := service.MyService.Docker().StartContainer(containerID); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	if err := service.MyService.Docker().RemoveContainer(id, true); err != nil {
		c.JSON(http.StatusInternalServerError, modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: err.Error()})
		return
	}

	delete(service.NewVersionApp, id)

	c.JSON(http.StatusOK, modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS)})
}

func GetDockerNetworks(c *gin.Context) {
	networks := service.MyService.Docker().GetNetworkList()
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
	info, err := service.MyService.Docker().DescribeContainer(appID)
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
func MyAppList(c *gin.Context) {
	index, _ := strconv.Atoi(c.DefaultQuery("index", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "0"))
	position, _ := strconv.ParseBool(c.DefaultQuery("position", "true"))
	list, unTranslation := service.MyService.Docker().GetContainerAppList(index, size, position)
	data := make(map[string]interface{}, 2)
	data["casaos_apps"] = list
	data["local_apps"] = unTranslation

	c.JSON(common_err.SUCCESS, &modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}

// @Summary my app hardware usage list
// @Produce  application/json
// @Accept application/json
// @Tags app
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/usage [get]
func AppUsageList(c *gin.Context) {
	list := service.MyService.Docker().GetContainerStats()
	c.JSON(common_err.SUCCESS, &modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: list})
	// c.JSON(common_err.SUCCESS, &modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: nil})
}

func GetDockerDaemonConfiguration(c *gin.Context) {
	// info, err := service.MyService.Docker().GetServerInfo()
	// if err != nil {
	// 	c.JSON(common_err.SERVICE_ERROR, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR), Data: err.Error()})
	// 	return
	// }
	data := make(map[string]interface{})

	if file.Exists(dockerRootDirFilePath) {
		buf := file.ReadFullFile(dockerRootDirFilePath)
		err := json.Unmarshal(buf, &data)
		if err != nil {
			c.JSON(common_err.CLIENT_ERROR, &modelCommon.Result{Success: common_err.CLIENT_ERROR, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: err})
			return
		}
	}
	c.JSON(common_err.SUCCESS, &modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
}

func PutDockerDaemonConfiguration(c *gin.Context) {
	request := make(map[string]interface{})
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, &modelCommon.Result{Success: common_err.CLIENT_ERROR, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: err})
		return
	}

	value, ok := request["docker_root_dir"]
	if !ok {
		c.JSON(http.StatusBadRequest, &modelCommon.Result{Success: common_err.CLIENT_ERROR, Message: common_err.GetMsg(common_err.INVALID_PARAMS), Data: "`docker_root_dir` should not empty"})
		return
	}

	dockerConfig := model.DockerDaemonConfigurationModel{}
	if file.Exists(dockerDaemonConfigurationFilePath) {
		byteResult := file.ReadFullFile(dockerDaemonConfigurationFilePath)
		err := json.Unmarshal(byteResult, &dockerConfig)
		if err != nil {
			c.JSON(http.StatusInternalServerError, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "error when trying to deserialize " + dockerDaemonConfigurationFilePath, Data: err})
			return
		}
	}

	dockerRootDir := value.(string)
	if dockerRootDir == "/" {
		dockerConfig.Root = "" // omitempty - empty string will not be serialized
	} else {
		if !file.Exists(dockerRootDir) {
			c.JSON(http.StatusBadRequest, &modelCommon.Result{Success: common_err.CLIENT_ERROR, Message: common_err.GetMsg(common_err.DIR_NOT_EXISTS), Data: common_err.GetMsg(common_err.DIR_NOT_EXISTS)})
			return
		}

		dockerConfig.Root = filepath.Join(dockerRootDir, "docker")

		if err := file.IsNotExistMkDir(dockerConfig.Root); err != nil {
			c.JSON(http.StatusInternalServerError, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "error when trying to create " + dockerConfig.Root, Data: err})
			return
		}
	}

	buf, err := json.Marshal(request)
	if err != nil {
		c.JSON(http.StatusBadRequest, &modelCommon.Result{Success: common_err.CLIENT_ERROR, Message: "error when trying to serialize docker root json", Data: err})
		return
	}

	if err := file.WriteToFullPath(buf, dockerRootDirFilePath, 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "error when trying to write " + dockerRootDirFilePath, Data: err})
		return
	}

	buf, err = json.Marshal(dockerConfig)
	if err != nil {
		c.JSON(http.StatusBadRequest, &modelCommon.Result{Success: common_err.CLIENT_ERROR, Message: "error when trying to serialize docker config", Data: dockerConfig})
		return
	}

	if err := file.WriteToFullPath(buf, dockerDaemonConfigurationFilePath, 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "error when trying to write to " + dockerDaemonConfigurationFilePath, Data: err})
		return
	}

	if err := systemctl.ReloadDaemon(); err != nil {
		c.JSON(http.StatusInternalServerError, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "error when trying to reload systemd daemon"})
	}

	if err := systemctl.StopService("docker"); err != nil {
		c.JSON(http.StatusInternalServerError, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "error when trying to stop docker service"})
	}

	if err := systemctl.StartService("docker"); err != nil {
		c.JSON(http.StatusInternalServerError, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: "error when trying to start docker service"})
	}

	c.JSON(http.StatusOK, &modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: request})
}
