package v1

import (
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	modelCommon "github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/port"
	"github.com/IceWhaleTech/CasaOS-Common/utils/systemctl"

	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
)

const (
	dockerRootDirFilePath             = "/var/lib/casaos/docker_root"
	dockerDaemonConfigurationFilePath = "/etc/docker/daemon.json"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// @Summary 获取远程列表
// @Produce  application/json
// @Accept application/json
// @Tags app
// @Param index query int false "页码"
// @Param size query int false "每页数量"
// @Param  category_id query int false "分类id"
// @Param  type query string false "rank,new"
// @Param  key query string false "search key"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/list [get]
func AppList(c *gin.Context) {
	index := c.DefaultQuery("index", "1")
	size := c.DefaultQuery("size", "10000")
	t := c.DefaultQuery("type", "rank")
	categoryID := c.DefaultQuery("category_id", "0")
	key := c.DefaultQuery("key", "")
	if len(index) == 0 || len(size) == 0 || len(t) == 0 || len(categoryID) == 0 {
		c.JSON(common_err.CLIENT_ERROR, &modelCommon.Result{Success: common_err.INVALID_PARAMS, Message: common_err.GetMsg(common_err.INVALID_PARAMS)})
		return
	}
	collection, err := service.MyService.App().GetServerList(index, size, t, categoryID, key)
	if err != nil {
		c.JSON(common_err.SERVICE_ERROR, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR), Data: err.Error()})
		return
	}

	data := make(map[string]interface{}, 3)
	data["recommend"] = collection.Recommend
	data["list"] = collection.List
	data["community"] = collection.Community

	c.JSON(common_err.SUCCESS, &modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: data})
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
	list, unTranslation := service.MyService.Docker().GetMyList(index, size, position)
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
	list := service.MyService.Docker().GetHardwareUsage()
	c.JSON(common_err.SUCCESS, &modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: list})
	// c.JSON(common_err.SUCCESS, &modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: nil})
}

// @Summary 应用详情
// @Produce  application/json
// @Accept application/json
// @Tags app
// @Param  id path int true "id"
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/appinfo/{id} [get]
func AppInfo(c *gin.Context) {
	id := c.Param("id")
	language := c.GetHeader("Language")
	info, err := service.MyService.App().GetServerAppInfo(id, "", language)
	if err != nil {
		c.JSON(common_err.SERVICE_ERROR, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR), Data: err.Error()})
		return
	}
	if info.NetworkModel != "host" {
		for i := 0; i < len(info.Ports); i++ {
			if p, _ := strconv.Atoi(info.Ports[i].ContainerPort); port.IsPortAvailable(p, info.Ports[i].Protocol) {
				info.Ports[i].CommendPort = strconv.Itoa(p)
			} else {
				if info.Ports[i].Protocol == "tcp" {
					if p, err := port.GetAvailablePort("tcp"); err == nil {
						info.Ports[i].CommendPort = strconv.Itoa(p)
					}
				} else if info.Ports[i].Protocol == "upd" {
					if p, err := port.GetAvailablePort("udp"); err == nil {
						info.Ports[i].CommendPort = strconv.Itoa(p)
					}
				}
			}

			if info.Ports[i].Type == 0 {
				info.PortMap = info.Ports[i].CommendPort
			}
		}
	} else {
		for i := 0; i < len(info.Ports); i++ {
			if info.Ports[i].Type == 0 {
				info.PortMap = info.Ports[i].ContainerPort
				break
			}
		}
	}

	for i := 0; i < len(info.Devices); i++ {
		if !file.CheckNotExist(info.Devices[i].ContainerPath) {
			info.Devices[i].Path = info.Devices[i].ContainerPath
		}
	}

	info.Image += ":" + info.ImageVersion

	c.JSON(common_err.SUCCESS, &modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: info})
}

// @Summary 获取远程分类列表
// @Produce  application/json
// @Accept application/json
// @Tags app
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/category [get]
func CategoryList(c *gin.Context) {
	list, err := service.MyService.App().GetServerCategoryList()
	if err != nil {
		c.JSON(common_err.SERVICE_ERROR, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR), Data: err.Error()})
		return
	}
	var count uint
	for _, category := range list {
		count += category.Count
	}

	rear := append([]model.CategoryList{}, list[0:]...)
	list = append(list[:0], model.CategoryList{Count: count, Name: "All", Font: "apps"})
	list = append(list, rear...)
	c.JSON(common_err.SUCCESS, &modelCommon.Result{Success: common_err.SUCCESS, Message: common_err.GetMsg(common_err.SUCCESS), Data: list})
}

// @Summary 分享该应用配置
// @Produce  application/json
// @Accept application/json
// @Tags app
// @Security ApiKeyAuth
// @Success 200 {string} string "ok"
// @Router /app/share [post]
func ShareAppFile(c *gin.Context) {
	str, _ := ioutil.ReadAll(c.Request.Body)
	content := service.MyService.App().ShareAppFile(str)
	c.JSON(common_err.SUCCESS, jsoniter.RawMessage(content))
}

func GetDockerDaemonConfiguration(c *gin.Context) {
	// info, err := service.MyService.Docker().GetDockerInfo()
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
