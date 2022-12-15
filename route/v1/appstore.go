package v1

import (
	"io/ioutil"
	"strconv"

	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	modelCommon "github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/port"

	"github.com/IceWhaleTech/CasaOS-AppManagement/service"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
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
	collection, err := service.MyService.AppStore().GetServerList(index, size, t, categoryID, key)
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
	info, err := service.MyService.AppStore().GetServerAppInfo(id, "", language)
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
	list, err := service.MyService.AppStore().GetServerCategoryList()
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
	content := service.MyService.AppStore().ShareAppFile(str)
	c.JSON(common_err.SUCCESS, jsoniter.RawMessage(content))
}
