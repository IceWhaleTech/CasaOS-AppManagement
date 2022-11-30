package service

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/http"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	client2 "github.com/docker/docker/client"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type AppService interface {
	GetServerList(index, size, tp, categoryID, key string) (model.ServerAppListCollection, error)
	GetServerAppInfo(id, t string, language string) (model.ServerAppList, error)
	GetServerCategoryList() (list []model.CategoryList, err error)
	AsyncGetServerCategoryList() ([]model.CategoryList, error)
	ShareAppFile(body []byte) string

	GetMyList(index, size int, position bool) (*[]model.MyAppList, *[]model.MyAppList)
	GetContainerInfo(id string) (types.Container, error)
	GetSimpleContainerInfo(id string) (types.Container, error)
	GetSystemAppList() []types.Container
	GetHardwareUsageStream()
	GetHardwareUsage() []model.DockerStatsModel
	GetAppStats(id string) string
}

type appStruct struct{}

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func (o *appStruct) GetServerList(index, size, tp, categoryID, key string) (model.ServerAppListCollection, error) {
	collection := model.ServerAppListCollection{}

	keyName := fmt.Sprintf("list_%s_%s_%s_%s_%s", index, size, tp, categoryID, "en")
	logger.Info("getting app list collection from cache...", zap.String("key", keyName))
	if result, ok := Cache.Get(keyName); ok {
		if collectionBytes, ok := result.([]byte); ok {
			if err := json.Unmarshal(collectionBytes, &collection); err != nil {
				logger.Error("error when deserializing app list collection from cache", zap.Any("err", err), zap.Any("content", collectionBytes))
				return collection, err
			}

			return collection, nil
		}
	}

	path := filepath.Join(config.AppInfo.DBPath, "/app_list.json")
	logger.Info("getting app list collection from local file...", zap.String("path", path))
	collectionBytes := file.ReadFullFile(path)
	if err := json.Unmarshal(collectionBytes, &collection); err != nil {
		logger.Info("app list collection from local file is either empty or broken - getting from online...", zap.String("path", path), zap.String("content", string(collectionBytes)))
		collection, err = o.AsyncGetServerList()
		if err != nil {
			return collection, err
		}
	}

	go func() {
		if _, err := o.AsyncGetServerList(); err != nil {
			logger.Error("error when getting app list collection from online", zap.Any("err", err))
		}
	}()

	if categoryID != "0" {
		categoryInt, _ := strconv.Atoi(categoryID)
		nList := []model.ServerAppList{}
		for _, v := range collection.List {
			if v.CategoryID == categoryInt {
				nList = append(nList, v)
			}
		}
		collection.List = nList
		nCommunity := []model.ServerAppList{}
		for _, v := range collection.Community {
			if v.CategoryID == categoryInt {
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

func (o *appStruct) AsyncGetServerList() (model.ServerAppListCollection, error) {
	collection := model.ServerAppListCollection{}

	path := filepath.Join(config.AppInfo.DBPath, "/app_list.json")

	logger.Info("getting app list collection from local file...", zap.String("path", path))
	collectionBytes := file.ReadFullFile(path)

	if err := json.Unmarshal(collectionBytes, &collection); err != nil {
		logger.Info("app list collection from local file is either empty or broken.", zap.String("path", path), zap.String("content", string(collectionBytes)))
	}

	headers := map[string]string{"Authorization": GetToken()}
	url := config.ServerInfo.ServerAPI + "/v2/app/newlist?index=1&size=1000&rank=name&category_id=0&key=&language=en"

	logger.Info("getting app list collection from online...", zap.String("url", url))
	resp, err := http.GetWithHeader(url, 30*time.Second, headers)
	if err != nil {
		logger.Error("error when calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", headers))
		return collection, err
	}
	defer resp.Body.Close()

	list, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("error when reading from response body after calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", headers))
		return collection, err
	}

	listModel := []model.ServerAppList{}
	communityModel := []model.ServerAppList{}
	recommendModel := []model.ServerAppList{}

	if err := json.Unmarshal([]byte(jsoniter.Get(list, "data", "list").ToString()), &listModel); err != nil {
		logger.Error("error when deserializing", zap.Any("err", err), zap.Any("content", string(list)), zap.Any("property", "data.list"))
		return collection, err
	}

	if err := json.Unmarshal([]byte(jsoniter.Get(list, "data", "recommend").ToString()), &recommendModel); err != nil {
		logger.Error("error when deserializing", zap.Any("err", err), zap.Any("content", string(list)), zap.Any("property", "data.recommend"))
		return collection, err
	}

	if err := json.Unmarshal([]byte(jsoniter.Get(list, "data", "community").ToString()), &communityModel); err != nil {
		logger.Error("error when deserializing", zap.Any("err", err), zap.Any("content", string(list)), zap.Any("property", "data.community"))
		return collection, err
	}

	if len(listModel) > 0 {
		collection.Community = communityModel
		collection.List = listModel
		collection.Recommend = recommendModel

		var by []byte
		by, err = json.Marshal(collection)
		if err != nil {
			logger.Error("marshal error", zap.Any("err", err))
		}

		if err := file.WriteToPath(by, config.AppInfo.DBPath, "app_list.json"); err != nil {
			logger.Error("error when writing to file", zap.Error(err), zap.Any("path", filepath.Join(config.AppInfo.DBPath, "app_list.json")))
		}
	}
	return collection, nil
}

func (o *appStruct) GetServerAppInfo(id, t string, language string) (model.ServerAppList, error) {
	head := make(map[string]string)

	head["Authorization"] = GetToken()

	info := model.ServerAppList{}

	url := config.ServerInfo.ServerAPI + "/v2/app/info/" + id + "?t=" + t + "&language=" + language
	resp, err := http.GetWithHeader(url, 30*time.Second, head)
	if err != nil {
		logger.Error("error when calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", head))
		return info, err
	}

	infoB, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("error when reading from response body after calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", head))
		return info, err
	}

	if len(infoB) == 0 {
		return info, errors.New("server error")
	}

	if err := json.Unmarshal([]byte(jsoniter.Get(infoB, "data").ToString()), &info); err != nil {
		fmt.Println(string(infoB))
		return info, err
	}

	return info, nil
}

func (o *appStruct) GetServerCategoryList() (list []model.CategoryList, err error) {
	category := model.ServerCategoryList{}
	results := file.ReadFullFile(config.AppInfo.DBPath + "/app_category.json")
	err = json.Unmarshal(results, &category)
	if err != nil {
		logger.Error("marshal error", zap.Any("err", err), zap.Any("content", string(results)))
		return o.AsyncGetServerCategoryList()
	}
	go func() {
		if _, err := o.AsyncGetServerCategoryList(); err != nil {
			logger.Error("error when getting server category list", zap.Error(err))
		}
	}()
	return category.Item, err
}

func (o *appStruct) AsyncGetServerCategoryList() ([]model.CategoryList, error) {
	list := model.ServerCategoryList{}
	results := file.ReadFullFile(config.AppInfo.DBPath + "/app_category.json")
	err := json.Unmarshal(results, &list)
	if err != nil {
		logger.Error("marshal error", zap.Any("err", err), zap.Any("content", string(results)))
	}
	item := []model.CategoryList{}
	head := make(map[string]string)
	head["Authorization"] = GetToken()

	url := config.ServerInfo.ServerAPI + "/v2/app/category"
	resp, err := http.GetWithHeader(url, 30*time.Second, head)
	if err != nil {
		logger.Error("error when calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", head))
		return item, err
	}

	listB, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("error when reading from response body after calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", head))
		return item, err
	}

	if len(listB) == 0 {
		return item, errors.New("server error")
	}

	if err := json.Unmarshal([]byte(jsoniter.Get(listB, "data").ToString()), &item); err != nil {
		logger.Error("error when deserializing", zap.Any("err", err), zap.String("content", string(listB)), zap.Any("property", "data"))
		return item, err
	}

	if len(item) > 0 {
		list.Item = item
		by, err := json.Marshal(list)
		if err != nil {
			logger.Error("marshal error", zap.Any("err", err))
		}
		if err := file.WriteToPath(by, config.AppInfo.DBPath, "app_category.json"); err != nil {
			logger.Error("error when writing to file", zap.Error(err), zap.Any("path", filepath.Join(config.AppInfo.DBPath, "app_category.json")))
		}
	}
	return item, nil
}

func (o *appStruct) ShareAppFile(body []byte) string {
	head := make(map[string]string)

	head["Authorization"] = GetToken()

	url := config.ServerInfo.ServerAPI + "/v1/community/add"
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

// 获取我的应用列表
func (o *appStruct) GetMyList(index, size int, position bool) (*[]model.MyAppList, *[]model.MyAppList) {
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

	unTranslation := []model.MyAppList{}

	list := []model.MyAppList{}

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

			list = append(list, model.MyAppList{
				Name:     name,
				Icon:     icon,
				State:    m.State,
				CustomID: m.Labels["custom_id"],
				ID:       m.ID,
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
			unTranslation = append(unTranslation, model.MyAppList{
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
			})
		}
	}

	return &list, &unTranslation
}

// system application list
func (o *appStruct) GetSystemAppList() []types.Container {
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

	return containers
}

// 获取我的应用列表
func (o *appStruct) GetContainerInfo(id string) (types.Container, error) {
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

func (o *appStruct) GetSimpleContainerInfo(id string) (types.Container, error) {
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

var dataStats sync.Map

var isFinish bool

func (o *appStruct) GetAppStats(id string) string {
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

func (o *appStruct) GetHardwareUsage() []model.DockerStatsModel {
	stream := true
	for !isFinish {
		if stream {
			stream = false
			go func() {
				o.GetHardwareUsageStream()
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

func (o *appStruct) GetHardwareUsageStream() {
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

func NewAppService() AppService {
	return &appStruct{}
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
		url := config.ServerInfo.ServerAPI + "/token"

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

		t <- jsoniter.Get(buf, "data").ToString()
	}()
	auth = <-t

	Cache.SetDefault(keyName, auth)
	return auth
}
