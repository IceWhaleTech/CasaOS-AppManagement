package v1

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/IceWhaleTech/CasaOS-AppManagement/model"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/docker"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	httpUtil "github.com/IceWhaleTech/CasaOS-Common/utils/http"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	jsoniter "github.com/json-iterator/go"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/zap"
)

type AppStore interface {
	GetServerList(index, size, tp, categoryID, key string) (*model.ServerAppListCollection, error)
	GetServerAppInfo(id, t string, language string) (model.ServerAppList, error)
	GetServerCategoryList() (list []model.CategoryList, err error)
	AsyncGetServerList(checkArchitectures bool) (*model.ServerAppListCollection, error)
	AsyncGetServerCategoryList() ([]model.CategoryList, error)
}

type appStore struct{}

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary

	Cache *cache.Cache

	mutex = sync.Mutex{}
)

func (o *appStore) GetServerList(index, size, tp, categoryID, key string) (*model.ServerAppListCollection, error) {
	collection := &model.ServerAppListCollection{}

	keyName := fmt.Sprintf("list_%s_%s_%s_%s_%s", index, size, tp, categoryID, "en")
	logger.Info("getting app list collection from cache...", zap.String("key", keyName))
	if result, ok := Cache.Get(keyName); ok {
		if collectionBytes, ok := result.([]byte); ok {
			if err := json.Unmarshal(collectionBytes, &collection); err != nil {
				logger.Error("error when deserializing app list collection from cache", zap.Any("err", err), zap.Any("content", collectionBytes))
				return nil, err
			}

			return collection, nil
		}
	}

	path := filepath.Join(config.AppInfo.DBPath, "/app_list.json")
	logger.Info("getting app list collection from local file...", zap.String("path", path))
	collectionBytes := file.ReadFullFile(path)
	if err := json.Unmarshal(collectionBytes, &collection); err != nil {
		logger.Info("app list collection from local file is either empty or broken - getting from online...", zap.String("path", path), zap.String("content", string(collectionBytes)))
		collection, err = o.AsyncGetServerList(false)
		if err != nil {
			return nil, err
		}
	}

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

func (o *appStore) AsyncGetServerList(checkArchitectures bool) (*model.ServerAppListCollection, error) {
	mutex.Lock()
	defer mutex.Unlock()

	collection := &model.ServerAppListCollection{}

	path := filepath.Join(config.AppInfo.DBPath, "/app_list.json")

	logger.Info("getting app list collection from local file...", zap.String("path", path))
	collectionBytes := file.ReadFullFile(path)

	if err := json.Unmarshal(collectionBytes, &collection); err != nil {
		logger.Info("app list collection from local file is either empty or broken.", zap.String("path", path), zap.String("content", string(collectionBytes)))
	}

	headers := map[string]string{"Authorization": GetToken()}
	url := config.ServerInfo.ServerAPI + "/v2/app/newlist?index=1&size=1000&rank=name&category_id=0&key=&language=en"

	logger.Info("getting app list collection from online...", zap.String("url", url))
	resp, err := httpUtil.GetWithHeader(url, 30*time.Second, headers)
	if err != nil {
		logger.Error("error when calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", headers))
		return collection, err
	}
	defer resp.Body.Close()

	list, err := io.ReadAll(resp.Body)
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

	if len(listModel) == 0 {
		return collection, nil
	}

	collection.Community = communityModel
	collection.List = listModel
	collection.Recommend = recommendModel

	if checkArchitectures {
		collection = updateArchitectures(collection)
	}

	var by []byte
	by, err = json.Marshal(collection)
	if err != nil {
		logger.Error("marshal error", zap.Any("err", err))
	}

	if err := file.WriteToPath(by, config.AppInfo.DBPath, "app_list.json"); err != nil {
		logger.Error("error when writing to file", zap.Error(err), zap.Any("path", filepath.Join(config.AppInfo.DBPath, "app_list.json")))
	}

	return collection, nil
}

func (o *appStore) GetServerAppInfo(id, t string, language string) (model.ServerAppList, error) {
	head := make(map[string]string)

	head["Authorization"] = GetToken()

	info := model.ServerAppList{}

	url := config.ServerInfo.ServerAPI + "/v2/app/info/" + id + "?t=" + t + "&language=" + language
	resp, err := httpUtil.GetWithHeader(url, 30*time.Second, head)
	if err != nil {
		logger.Error("error when calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", head))
		return info, err
	}

	infoB, err := io.ReadAll(resp.Body)
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

func (o *appStore) GetServerCategoryList() (list []model.CategoryList, err error) {
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

func (o *appStore) AsyncGetServerCategoryList() ([]model.CategoryList, error) {
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
	resp, err := httpUtil.GetWithHeader(url, 30*time.Second, head)
	if err != nil {
		logger.Error("error when calling url with header", zap.Any("err", err), zap.Any("url", url), zap.Any("head", head))
		return item, err
	}

	listB, err := io.ReadAll(resp.Body)
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

func NewAppService() AppStore {
	return &appStore{}
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

		resp, err := httpUtil.Get(url, 30*time.Second)
		if err != nil {
			logger.Error("error when calling url", zap.Any("err", err), zap.Any("url", url))
			t <- ""
			return
		}

		buf, err := io.ReadAll(resp.Body)
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

func updateArchitectures(collection *model.ServerAppListCollection) *model.ServerAppListCollection {
	result := model.ServerAppListCollection{
		List:      make([]model.ServerAppList, len(collection.List)),
		Recommend: make([]model.ServerAppList, len(collection.Recommend)),
		Community: make([]model.ServerAppList, len(collection.Community)),
	}

	p := pool.New().WithMaxGoroutines(3)

	for i, app := range collection.List {
		p.Go(func() {
			result.List[i] = app

			archs, err := getArchitectures(fmt.Sprintf("%s:%s", app.Image, app.ImageVersion))
			if err != nil {
				logger.Error("error when getting architectures for apps in list", zap.Error(err), zap.Any("image", app.Image))
				return
			}
			result.List[i].Architectures = archs
		})
	}

	for i, app := range collection.Recommend {
		p.Go(func() {
			result.Recommend[i] = app
			archs, err := getArchitectures(app.Image)
			if err != nil {
				logger.Error("error when getting architectures for apps in recommend list", zap.Error(err), zap.Any("image", app.Image))
				return
			}
			result.Recommend[i].Architectures = archs
		})
	}

	for i, app := range collection.Community {
		p.Go(func() {
			result.Community[i] = app
			archs, err := getArchitectures(app.Image)
			if err != nil {
				logger.Error("error when getting architectures for apps in community list", zap.Error(err), zap.Any("image", app.Image))
				return
			}
			result.Community[i].Architectures = archs
		})
	}

	p.Wait()

	return &result
}

func getArchitectures(imageName string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	index, err := docker.RemoteManifest(ctx, imageName)
	if err != nil {
		return nil, err
	}
	architectures := []string{}
	for _, platform := range index.Manifests {
		architectures = append(architectures, platform.Platform.Architecture)
	}

	return architectures, nil
}
