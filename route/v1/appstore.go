package v1

import (
	"strconv"
	"strings"

	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	modelCommon "github.com/IceWhaleTech/CasaOS-Common/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/common_err"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/port"
	"github.com/samber/lo"

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

	serverAppLists, err := service.MyService.V1AppStore().GetServerList(index, size, t, categoryID, key)
	if err != nil {
		c.JSON(common_err.SERVICE_ERROR, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR), Data: err.Error()})
		return
	}

	myAppList, _ := service.MyService.Docker().GetContainerAppList(nil, nil, nil)

	data := make(map[string]interface{}, 3)
	data["recommend"] = updateState(&serverAppLists.Recommend, myAppList)
	data["list"] = updateState(&serverAppLists.List, myAppList)
	data["community"] = updateState(&serverAppLists.Community, myAppList)

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
	info, err := service.MyService.V1AppStore().GetServerAppInfo(id, "", language)
	if err != nil {
		c.JSON(common_err.SERVICE_ERROR, &modelCommon.Result{Success: common_err.SERVICE_ERROR, Message: common_err.GetMsg(common_err.SERVICE_ERROR), Data: err.Error()})
		return
	}
	if info.NetworkModel != "host" {
		for i := 0; i < len(info.Ports); i++ {
			protocol := strings.ToLower(info.Ports[i].Protocol)

			if p, _ := strconv.Atoi(info.Ports[i].ContainerPort); port.IsPortAvailable(p, protocol) {
				info.Ports[i].CommendPort = strconv.Itoa(p)
			} else {
				if protocol == "tcp" {
					if p, err := port.GetAvailablePort("tcp"); err == nil {
						info.Ports[i].CommendPort = strconv.Itoa(p)
					}
				} else if protocol == "upd" {
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

	myAppList, _ := service.MyService.Docker().GetContainerAppList(nil, &strings.Split(info.Image, ":")[0], nil)
	if len(*myAppList) > 0 {
		info.State = model.StateEnumInstalled
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
	list, err := service.MyService.V1AppStore().GetServerCategoryList()
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

func updateState(serverAppList *[]model.ServerAppList, myAppList *[]model.MyAppList) []model.ServerAppList {
	result := make([]model.ServerAppList, len(*serverAppList))
	for i, serverApp := range *serverAppList {
		if lo.ContainsBy(*myAppList, func(app model.MyAppList) bool {
			return serverApp.Image == strings.Split(app.Image, ":")[0]
		}) {
			serverApp.State = model.StateEnumInstalled
		}
		result[i] = serverApp
	}

	return result
}
