package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	model2 "github.com/IceWhaleTech/CasaOS-AppManagement/service/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/http"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	client2 "github.com/docker/docker/client"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AppService interface {
	GetServerList(index, size, tp, categoryId, key string) (model.ServerAppListCollection, error)
	GetServerAppInfo(id, t string, language string) (model.ServerAppList, error)
	GetServerCategoryList() (list []model.CategoryList, err error)
	AsyncGetServerCategoryList() ([]model.CategoryList, error)
	ShareAppFile(body []byte) string

	GetMyList(index, size int, position bool) (*[]model2.MyAppList, *[]model2.MyAppList)
	SaveContainer(m model2.AppListDBModel)
	GetUninstallInfo(id string) model2.AppListDBModel
	DeleteApp(id string)
	GetContainerInfo(id string) (types.Container, error)
	GetAppDBInfo(id string) model2.AppListDBModel
	UpdateApp(m model2.AppListDBModel)
	GetSimpleContainerInfo(id string) (types.Container, error)
	DelAppConfigDir(path string)
	GetSystemAppList() []types.Container
	GetHardwareUsageStream()
	GetHardwareUsage() []model.DockerStatsModel
	GetAppStats(id string) string
	GetAllDBApps() []model2.AppListDBModel
	ImportApplications(casaApp bool)
	CheckNewImage()
}

type appStruct struct {
	db *gorm.DB
}

var json2 = jsoniter.ConfigCompatibleWithStandardLibrary

func (o *appStruct) GetServerList(index, size, tp, categoryId, key string) (model.ServerAppListCollection, error) {
	keyName := fmt.Sprintf("list_%s_%s_%s_%s_%s", index, size, tp, categoryId, "en")
	collection := model.ServerAppListCollection{}
	if result, ok := Cache.Get(keyName); ok {
		res, ok := result.(string)
		if ok {
			json2.Unmarshal([]byte(res), &collection)
			return collection, nil
		}
	}

	collectionStr := file.ReadFullFile(config.AppInfo.DBPath + "/app_list.json")

	err := json2.Unmarshal(collectionStr, &collection)
	if err != nil {
		logger.Error("marshal error", zap.Any("err", err), zap.Any("content", string(collectionStr)))
		collection, err = o.AsyncGetServerList()
		if err != nil {
			return collection, err
		}
	}

	go o.AsyncGetServerList()

	if categoryId != "0" {
		categoryInt, _ := strconv.Atoi(categoryId)
		nList := []model.ServerAppList{}
		for _, v := range collection.List {
			if v.CategoryId == categoryInt {
				nList = append(nList, v)
			}
		}
		collection.List = nList
		nCommunity := []model.ServerAppList{}
		for _, v := range collection.Community {
			if v.CategoryId == categoryInt {
				nCommunity = append(nCommunity, v)
			}
		}
		collection.Community = nCommunity
	}
	if tp != "name" {
		if tp == "new" {
			sort.Slice(collection.List, func(i, j int) bool {
				return collection.List[i].CreatedAt.After(collection.List[j].CreatedAt)
			})
			sort.Slice(collection.Community, func(i, j int) bool {
				return collection.Community[i].CreatedAt.After(collection.Community[j].CreatedAt)
			})
		} else if tp == "rank" {
			sort.Slice(collection.List, func(i, j int) bool {
				return collection.List[i].QueryCount > collection.List[j].QueryCount
			})
			sort.Slice(collection.Community, func(i, j int) bool {
				return collection.Community[i].QueryCount > collection.Community[j].QueryCount
			})
		}
	}
	sizeInt, _ := strconv.Atoi(size)

	if index != "1" {
		indexInt, _ := strconv.Atoi(index)
		collection.List = collection.List[(indexInt-1)*sizeInt : indexInt*sizeInt]
		collection.Community = collection.Community[(indexInt-1)*sizeInt : indexInt*sizeInt]
	} else {
		if len(collection.List) > sizeInt {
			collection.List = collection.List[:sizeInt]
		}
		if len(collection.Community) > sizeInt {
			collection.Community = collection.Community[:sizeInt]
		}
	}

	if len(collection.List) > 0 {
		by, _ := json.Marshal(collection)
		Cache.Set(keyName, string(by), time.Minute*10)
	}

	return collection, nil
}

func (o *appStruct) AsyncGetServerList() (collection model.ServerAppListCollection, err error) {
	results := file.ReadFullFile(config.AppInfo.DBPath + "/app_list.json")
	errr := json2.Unmarshal(results, &collection)
	if errr != nil {
		logger.Error("marshal error", zap.Any("err", err), zap.Any("content", string(results)))
		return collection, errr
	}

	head := make(map[string]string)

	head["Authorization"] = GetToken()

	url := config.ServerInfo.ServerApi + "/v2/app/newlist?index=1&size=1000&rank=name&category_id=0&key=&language=en"
	resp, err := http.GetWitHeader(url, 30*time.Second, head)
	if err != nil {
		logger.Error("error when calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", head))
		return collection, err
	}

	list, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("error when reading from response body after calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", head))
		return collection, err
	}

	listS := string(list)

	listModel := []model.ServerAppList{}
	communityModel := []model.ServerAppList{}
	recommendModel := []model.ServerAppList{}
	err = json2.Unmarshal([]byte(gjson.Get(listS, "data.list").String()), &listModel)
	json2.Unmarshal([]byte(gjson.Get(listS, "data.recommend").String()), &recommendModel)
	json2.Unmarshal([]byte(gjson.Get(listS, "data.community").String()), &communityModel)

	if len(listModel) > 0 {
		collection.Community = communityModel
		collection.List = listModel
		collection.Recommend = recommendModel
		// TODO - collection.Version = o.GetCasaosVersion().Version
		var by []byte
		by, err = json.Marshal(collection)
		if err != nil {
			logger.Error("marshal error", zap.Any("err", err))
		}
		file.WriteToPath(by, config.AppInfo.DBPath, "app_list.json")
	}
	return
}

func (o *appStruct) GetServerAppInfo(id, t string, language string) (model.ServerAppList, error) {
	head := make(map[string]string)

	head["Authorization"] = GetToken()

	info := model.ServerAppList{}

	url := config.ServerInfo.ServerApi + "/v2/app/info/" + id + "?t=" + t + "&language=" + language
	resp, err := http.GetWitHeader(url, 30*time.Second, head)
	if err != nil {
		logger.Error("error when calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", head))
		return info, err
	}

	infoB, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("error when reading from response body after calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", head))
		return info, err
	}

	infoS := string(infoB)

	if infoS == "" {
		return info, errors.New("server error")
	}

	if err := json2.Unmarshal([]byte(gjson.Get(infoS, "data").String()), &info); err != nil {
		fmt.Println(infoS)
		return info, err
	}

	return info, nil
}

func (o *appStruct) GetServerCategoryList() (list []model.CategoryList, err error) {
	category := model.ServerCategoryList{}
	results := file.ReadFullFile(config.AppInfo.DBPath + "/app_category.json")
	err = json2.Unmarshal(results, &category)
	if err != nil {
		logger.Error("marshal error", zap.Any("err", err), zap.Any("content", string(results)))
		return o.AsyncGetServerCategoryList()
	}
	go o.AsyncGetServerCategoryList()
	return category.Item, err
}

func (o *appStruct) AsyncGetServerCategoryList() ([]model.CategoryList, error) {
	list := model.ServerCategoryList{}
	results := file.ReadFullFile(config.AppInfo.DBPath + "/app_category.json")
	err := json2.Unmarshal(results, &list)
	if err != nil {
		logger.Error("marshal error", zap.Any("err", err), zap.Any("content", string(results)))
	}
	item := []model.CategoryList{}
	head := make(map[string]string)
	head["Authorization"] = GetToken()

	url := config.ServerInfo.ServerApi + "/v2/app/category"
	resp, err := http.GetWitHeader(url, 30*time.Second, head)
	if err != nil {
		logger.Error("error when calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", head))
		return item, err
	}

	listB, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("error when reading from response body after calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", head))
		return item, err
	}

	listS := string(listB)
	if len(listS) == 0 {
		return item, errors.New("server error")
	}

	json2.Unmarshal([]byte(gjson.Get(listS, "data").String()), &item)
	if len(item) > 0 {
		// TODO - list.Version = o.GetCasaosVersion().Version
		list.Item = item
		by, err := json.Marshal(list)
		if err != nil {
			logger.Error("marshal error", zap.Any("err", err))
		}
		file.WriteToPath(by, config.AppInfo.DBPath, "app_category.json")
	}
	return item, nil
}

func (o *appStruct) ShareAppFile(body []byte) string {
	head := make(map[string]string)

	head["Authorization"] = GetToken()

	url := config.ServerInfo.ServerApi + "/v1/community/add"
	resp, err := http.PostWithHeader(url, body, 30*time.Second, head)
	if err != nil {
		logger.Error("error when calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", head))
		return ""
	}

	contentB, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("error when reading from response body after calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", head))
		return ""
	}

	content := string(contentB)
	return content
}

func (a *appStruct) CheckNewImage() {
	list := MyService.Docker().DockerContainerList()
	for _, v := range list {
		inspect, err := MyService.Docker().DockerImageInfo(strings.Split(v.Image, ":")[0])
		if err != nil {
			NewVersionApp[v.ID] = inspect.ID
			continue
		}
		if inspect.ID == v.ImageID {
			delete(NewVersionApp, v.ID)
			continue
		}
		NewVersionApp[v.ID] = inspect.ID
	}
}

func (a *appStruct) ImportApplications(casaApp bool) {
	if casaApp {
		list := MyService.App().GetAllDBApps()
		for _, app := range list {
			info, err := MyService.Docker().DockerContainerInfo(app.CustomId)
			if err != nil {
				MyService.App().DeleteApp(app.CustomId)
				continue
			}
			// info.NetworkSettings
			info.Config.Labels["casaos"] = "casaos"
			info.Config.Labels["web"] = app.PortMap
			info.Config.Labels["icon"] = app.Icon
			info.Config.Labels["desc"] = app.Description
			info.Config.Labels["index"] = app.Index
			info.Config.Labels["custom_id"] = app.CustomId
			info.Name = app.Title
			container_id, err := MyService.Docker().DockerContainerCopyCreate(info)
			if err != nil {
				fmt.Println(err)
				continue
			}
			MyService.App().DeleteApp(app.CustomId)
			MyService.Docker().DockerContainerStop(app.CustomId)
			MyService.Docker().DockerContainerRemove(app.CustomId, false)
			MyService.Docker().DockerContainerStart(container_id)

		}
	} else {
		list := MyService.Docker().DockerContainerList()
		for _, app := range list {
			info, err := MyService.Docker().DockerContainerInfo(app.ID)
			if err != nil || info.Config.Labels["casaos"] == "casaos" {
				continue
			}
			info.Config.Labels["casaos"] = "casaos"
			info.Config.Labels["web"] = ""
			info.Config.Labels["icon"] = ""
			info.Config.Labels["desc"] = ""
			info.Config.Labels["index"] = ""
			info.Config.Labels["custom_id"] = uuid.NewV4().String()

			_, err = MyService.Docker().DockerContainerCopyCreate(info)
			if err != nil {
				continue
			}

		}
	}

	// allcontainer := MyService.Docker().DockerContainerList()
	// for _, app := range allcontainer {
	// 	info, err := MyService.Docker().DockerContainerInfo(app.ID)
	// 	if err != nil {
	// 		continue
	// 	}
	// 	MyService.Docker().DockerContainerStop(app.ID)
	// 	MyService.Docker().DockerContainerRemove(app.ID, false)
	// 	//info.NetworkSettings
	// 	info.Config.Labels["custom_id"] = uuid.NewV4().String()
	// 	container_id, err := MyService.Docker().DockerContainerCopyCreate(info)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		continue
	// 	}
	// 	MyService.Docker().DockerContainerStart(container_id)
	//}
}

// 获取我的应用列表
func (a *appStruct) GetMyList(index, size int, position bool) (*[]model2.MyAppList, *[]model2.MyAppList) {
	cli, err := client2.NewClientWithOpts(client2.FromEnv, client2.WithTimeout(time.Second*5))
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

	unTranslation := []model2.MyAppList{}

	list := []model2.MyAppList{}

	for _, m := range containers {
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

			list = append(list, model2.MyAppList{
				Name:     name,
				Icon:     icon,
				State:    m.State,
				CustomId: m.Labels["custom_id"],
				Id:       m.ID,
				Port:     m.Labels["web"],
				Index:    m.Labels["index"],
				// Order:      m.Labels["order"],
				Image:  m.Image,
				Latest: newVersion,
				// Type:   m.Labels["origin"],
				// Slogan: m.Slogan,
				// Rely:     m.Rely,
				Host:     m.Labels["host"],
				Protocol: m.Labels["protocol"],
			})
		} else {
			unTranslation = append(unTranslation, model2.MyAppList{
				Name:     strings.ReplaceAll(m.Names[0], "/", ""),
				Icon:     "",
				State:    m.State,
				CustomId: m.ID,
				Id:       m.ID,
				Port:     "",
				Latest:   false,
				Host:     "",
				Protocol: "",
				Image:    m.Image,
			})
		}
	}

	// lMap := make(map[string]interface{})
	// for _, dbModel := range lm {
	// 	if position {
	// 		if dbModel.Position {
	// 			lMap[dbModel.ContainerId] = dbModel
	// 		}
	// 	} else {
	// 		lMap[dbModel.ContainerId] = dbModel
	// 	}
	// }
	// for _, container := range containers {

	// 	if lMap[container.ID] != nil && container.Labels["origin"] != "system" {
	// 		m := lMap[container.ID].(model2.AppListDBModel)
	// 		if len(m.Label) == 0 {
	// 			m.Label = m.Title
	// 		}

	// 		// info, err := cli.ContainerInspect(context.Background(), container.ID)
	// 		// var tm string
	// 		// if err != nil {
	// 		// 	tm = time.Now().String()
	// 		// } else {
	// 		// 	tm = info.State.StartedAt
	// 		//}
	// 		list = append(list, model2.MyAppList{
	// 			Name:     m.Label,
	// 			Icon:     m.Icon,
	// 			State:    container.State,
	// 			CustomId: strings.ReplaceAll(container.Names[0], "/", ""),
	// 			Port:     m.PortMap,
	// 			Index:    m.Index,
	// 			//UpTime:   tm,
	// 			Image:  m.Image,
	// 			Slogan: m.Slogan,
	// 			//Rely:     m.Rely,
	// 		})
	// 	}

	// }

	return &list, &unTranslation
}

// system application list
func (a *appStruct) GetSystemAppList() []types.Container {
	// 获取docker应用
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		logger.Error("Failed to init client", zap.Any("err", err))
	}
	defer cli.Close()
	fts := filters.NewArgs()
	fts.Add("label", "origin=system")
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true, Filters: fts})
	if err != nil {
		logger.Error("Failed to get container_list", zap.Any("err", err))
	}

	// 获取本地数据库应用

	// var lm []model2.AppListDBModel
	// a.db.Table(model2.CONTAINERTABLENAME).Select("title,icon,port_map,`index`,container_id,position,label,slogan,image,volumes").Find(&lm)

	// list := []model2.MyAppList{}
	// lMap := make(map[string]interface{})
	// for _, dbModel := range lm {
	// 	lMap[dbModel.ContainerId] = dbModel
	// }

	return containers
}

func (a *appStruct) GetAllDBApps() []model2.AppListDBModel {
	var lm []model2.AppListDBModel
	a.db.Table(model2.CONTAINERTABLENAME).Select("custom_id,title,icon,container_id,label,slogan,image,port_map").Find(&lm)
	return lm
}

// 获取我的应用列表
func (a *appStruct) GetContainerInfo(id string) (types.Container, error) {
	// 获取docker应用
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		logger.Error("Failed to init client", zap.Any("err", err))
	}
	filters := filters.NewArgs()
	filters.Add("id", id)
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		logger.Error("Failed to get container_list", zap.Any("err", err))
	}

	if len(containers) > 0 {
		return containers[0], nil
	}
	return types.Container{}, nil
}

func (a *appStruct) GetSimpleContainerInfo(id string) (types.Container, error) {
	// 获取docker应用
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return types.Container{}, err
	}
	defer cli.Close()
	filters := filters.NewArgs()
	filters.Add("id", id)
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		return types.Container{}, err
	}

	if len(containers) > 0 {
		return containers[0], nil
	}
	return types.Container{}, errors.New("container not existent")
}

// 获取我的应用列表
func (a *appStruct) GetAppDBInfo(id string) model2.AppListDBModel {
	var m model2.AppListDBModel
	a.db.Table(model2.CONTAINERTABLENAME).Where("custom_id = ?", id).First(&m)
	return m
}

// 根据容器id获取镜像名称
func (a *appStruct) GetUninstallInfo(id string) model2.AppListDBModel {
	var m model2.AppListDBModel
	a.db.Table(model2.CONTAINERTABLENAME).Select("image,version,enable_upnp,ports,envs,volumes,origin").Where("custom_id = ?", id).First(&m)
	return m
}

// 创建容器成功后保存容器
func (a *appStruct) SaveContainer(m model2.AppListDBModel) {
	a.db.Table(model2.CONTAINERTABLENAME).Create(&m)
}

func (a *appStruct) UpdateApp(m model2.AppListDBModel) {
	a.db.Table(model2.CONTAINERTABLENAME).Save(&m)
}

func (a *appStruct) DelAppConfigDir(path string) {
	// TODO - command.OnlyExec("source " + config.AppInfo.ShellPath + "/helper.sh ;DelAppConfigDir " + path)
}

func (a *appStruct) DeleteApp(id string) {
	a.db.Table(model2.CONTAINERTABLENAME).Where("custom_id = ?", id).Delete(&model2.AppListDBModel{})
}

var dataStats sync.Map

var isFinish bool = false

func (a *appStruct) GetAppStats(id string) string {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return ""
	}
	defer cli.Close()
	con, err := cli.ContainerStats(context.Background(), id, false)
	if err != nil {
		return err.Error()
	}
	defer con.Body.Close()
	c, _ := ioutil.ReadAll(con.Body)
	return string(c)
}

func (a *appStruct) GetHardwareUsage() []model.DockerStatsModel {
	stream := true
	for !isFinish {
		if stream {
			stream = false
			go func() {
				a.GetHardwareUsageStream()
			}()
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

func (a *appStruct) GetHardwareUsageStream() {
	cli, err := client2.NewClientWithOpts(client2.FromEnv)
	if err != nil {
		return
	}
	defer cli.Close()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	fts := filters.NewArgs()
	fts.Add("label", "casaos=casaos")
	// fts.Add("status", "running")
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true, Filters: fts})
	if err != nil {
		logger.Error("Failed to get container_list", zap.Any("err", err))
	}
	for i := 0; i < 100; i++ {
		if i%10 == 0 {
			containers, err = cli.ContainerList(context.Background(), types.ContainerListOptions{All: true, Filters: fts})
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
				decode := json.NewDecoder(stats.Body)
				var data interface{}
				if err := decode.Decode(&data); err == io.EOF {
					return
				}
				m, _ := dataStats.Load(v.ID)
				dockerStats := model.DockerStatsModel{}
				if m != nil {
					dockerStats.Previous = m.(model.DockerStatsModel).Data
				}
				dockerStats.Data = data
				dockerStats.Icon = v.Labels["icon"]
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
		dataStats = temp
		isFinish = true

		time.Sleep(time.Second * 1)
	}
	isFinish = false
	cancel()
}

func NewAppService(db *gorm.DB) AppService {
	return &appStruct{db: db}
}

func GetToken() string {
	t := make(chan string)
	keyName := "casa_token"

	var auth string
	if result, ok := Cache.Get(keyName); ok {
		auth, ok = result.(string)
		if ok {
			return auth
		}
	}
	go func() {
		url := config.ServerInfo.ServerApi + "/token"

		resp, err := http.Get(url, 30*time.Second)
		if err != nil {
			logger.Error("error when calling url", zap.Any("err", err), zap.Any("url", url))
			t <- ""
			return
		}

		buf, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Error("error when reading from response body after calling url", zap.Any("err", err), zap.Any("url", url))
			t <- ""
			return
		}

		t <- gjson.Get(string(buf), "data").String()
	}()
	auth = <-t

	Cache.SetDefault(keyName, auth)
	return auth
}
