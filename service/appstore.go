package service

import (
	"crypto/md5" // nolint: gosec
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IceWhaleTech/CasaOS-AppManagement/codegen"
	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/utils/downloadHelper"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type AppStore interface {
	Catalog() (map[string]*ComposeApp, error)
	CategoryMap() (map[string]codegen.CategoryInfo, error)
	ComposeApp(id string) (*ComposeApp, error)
	Recommend() ([]string, error)
	UpdateCatalog() error
	WorkDir() (string, error)
}

type appStore struct {
	categoryMap map[string]codegen.CategoryInfo
	catalog     map[string]*ComposeApp
	recommend   []string
	url         string

	lastAPPStoreSize int64
}

var (
	appStoreMap = make(map[string]*appStore)

	ErrNotAppStore             = fmt.Errorf("not an appstore")
	ErrDefaultAppStoreNotFound = fmt.Errorf("default appstore not found")
)

func (s *appStore) CategoryMap() (map[string]codegen.CategoryInfo, error) {
	if s.categoryMap != nil {
		return s.categoryMap, nil
	}

	workdir, err := s.WorkDir()
	if err != nil {
		return nil, err
	}

	storeRoot, err := StoreRoot(workdir)
	if err != nil {
		return nil, err
	}

	categoryMap := LoadCategoryMap(storeRoot)

	s.categoryMap = categoryMap

	return s.categoryMap, nil
}

func (s *appStore) UpdateCatalog() error {
	isSuccessful := false

	if _, err := url.Parse(s.url); err != nil {
		return err
	}

	// check wether the zip package size change
	// if not, skip the update
	{
		// timeout 5s
		http.DefaultClient.Timeout = 5 * time.Second
		res, err := http.Head(s.url)
		if err != nil {
			return err
		}
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get appstore size, status code: %d", res.StatusCode)
		}
		if res.ContentLength == s.lastAPPStoreSize {
			logger.Info("appstore size not changed", zap.String("url", s.url))
			return nil
		}
		logger.Info("appstore size changed, update app store", zap.String("url", s.url))

		defer func() {
			if isSuccessful {
				s.lastAPPStoreSize = res.ContentLength
			}
		}()
	}

	workdir, err := s.WorkDir()
	if err != nil {
		return err
	}
	tmpDir := workdir + ".tmp"

	defer func() {
		if err := file.RMDir(tmpDir); err != nil {
			logger.Error("failed to remove temp appstore workdir", zap.Error(err), zap.String("tmpDir", tmpDir))
		}
	}()

	if err := downloadHelper.Download(s.url, tmpDir); err != nil {
		return err
	}

	// make a backup of existing workdir
	if file.Exists(workdir) {
		backupDir := workdir + ".backup"

		if err := file.RMDir(backupDir); err != nil {
			return err
		}

		if err := os.Rename(workdir, backupDir); err != nil {
			return err
		}

		defer func() {
			if isSuccessful {
				if err := file.RMDir(backupDir); err != nil {
					logger.Error("failed to remove backup appstore workdir", zap.Error(err), zap.String("backupDir", backupDir))
				}
				return
			}

			if err := file.RMDir(workdir); err != nil {
				logger.Error("failed to remove appstore workdir", zap.Error(err), zap.String("workdir", workdir))
			}

			if err := os.Rename(backupDir, workdir); err != nil {
				logger.Error("failed to restore backup appstore workdir", zap.Error(err), zap.String("backupDir", backupDir), zap.String("workdir", workdir))
			}
		}()
	}

	if err := os.Rename(tmpDir, workdir); err != nil {
		return err
	}

	storeRoot, err := StoreRoot(workdir)
	if err != nil {
		return err
	}

	placeholderFile := filepath.Join(storeRoot, ".casaos-appstore")
	if err := file.CreateFileAndWriteContent(placeholderFile, s.url); err != nil {
		return err
	}

	s.catalog, err = BuildCatalog(storeRoot)
	if err != nil {
		return err
	}

	s.categoryMap = LoadCategoryMap(storeRoot)

	s.recommend = LoadRecommend(storeRoot)

	isSuccessful = true

	return nil
}

func (s *appStore) Recommend() ([]string, error) {
	if s.recommend != nil && len(s.recommend) > 0 {
		return s.recommend, nil
	}

	workdir, err := s.WorkDir()
	if err != nil {
		return nil, err
	}

	storeRoot, err := StoreRoot(workdir)
	if err != nil {
		return nil, err
	}

	return LoadRecommend(storeRoot), nil
}

func (s *appStore) Catalog() (map[string]*ComposeApp, error) {
	if s.catalog != nil && len(s.catalog) > 0 {
		return s.catalog, nil
	}

	workdir, err := s.WorkDir()
	if err != nil {
		return nil, err
	}

	storeRoot, err := StoreRoot(workdir)
	if err != nil {
		return nil, err
	}

	catalog, err := BuildCatalog(storeRoot)
	if err != nil {
		return nil, err
	}

	s.catalog = catalog

	return s.catalog, nil
}

func (s *appStore) ComposeApp(appStoreID string) (*ComposeApp, error) {
	catalog, err := s.Catalog()
	if err != nil {
		return nil, err
	}

	if composeApp, ok := catalog[appStoreID]; ok {
		return composeApp, nil
	}

	return nil, nil
}

func (s *appStore) WorkDir() (string, error) {
	if s.url == "default" {
		return filepath.Join(config.AppInfo.AppStorePath, s.url), nil
	}

	parsedURL, err := url.Parse(s.url)
	if err != nil {
		return "", err
	}

	appstoreKey := strings.ToLower(parsedURL.Path)

	hash := fmt.Sprintf("%x", md5.Sum([]byte(appstoreKey))) //nolint: gosec

	return filepath.Join(config.AppInfo.AppStorePath, parsedURL.Host, hash), nil
}

func AppStoreByURL(appstoreURL string) (AppStore, error) {
	_, err := url.Parse(appstoreURL)
	if err != nil {
		return nil, err
	}

	// a appstoreKey is a normalized appstore url where everything is in lowercase
	appstoreKey := strings.ToLower(appstoreURL)
	if appstore, ok := appStoreMap[appstoreKey]; ok {
		return appstore, nil
	}

	appStoreMap[appstoreKey] = &appStore{
		url:     appstoreURL,
		catalog: map[string]*ComposeApp{},
	}

	return appStoreMap[appstoreKey], nil
}

func NewDefaultAppStore() (AppStore, error) {
	storeRoot := filepath.Join(config.AppInfo.AppStorePath, "default")

	if !file.Exists(storeRoot) {
		return nil, ErrDefaultAppStoreNotFound
	}

	categoryMap := LoadCategoryMap(storeRoot)

	catalog, err := BuildCatalog(storeRoot)
	if err != nil {
		return nil, err
	}

	recommend := LoadRecommend(storeRoot)

	return &appStore{
		url:         "default",
		categoryMap: categoryMap,
		catalog:     catalog,
		recommend:   recommend,
	}, nil
}

func LoadCategoryMap(storeRoot string) map[string]codegen.CategoryInfo {
	categoryListFile := filepath.Join(storeRoot, common.CategoryListFileName)

	// unmarsal category list
	categoryList := []codegen.CategoryInfo{}

	if !file.Exists(categoryListFile) {
		return map[string]codegen.CategoryInfo{}
	}

	buf := file.ReadFullFile(categoryListFile)

	if err := json.Unmarshal(buf, &categoryList); err != nil {
		logger.Error("failed to unmarshal category list", zap.Error(err), zap.String("categoryListFile", categoryListFile))
		return map[string]codegen.CategoryInfo{}
	}

	categoryList = lo.Filter(categoryList, func(category codegen.CategoryInfo, i int) bool {
		return category.Name != nil && *category.Name != ""
	})

	categoryList = lo.Map(categoryList, func(category codegen.CategoryInfo, i int) codegen.CategoryInfo {
		if category.Font == nil || *category.Font == "" {
			category.Font = lo.ToPtr(common.DefaultCategoryFont)
		}

		if category.Description == nil {
			category.Description = lo.ToPtr("")
		}

		return category
	})

	return lo.SliceToMap(categoryList, func(category codegen.CategoryInfo) (string, codegen.CategoryInfo) {
		return *category.Name, category
	})
}

func LoadRecommend(storeRoot string) []string {
	recommendListFile := filepath.Join(storeRoot, common.RecommendListFileName)

	// unmarsal recommend list
	recommendList := []interface{}{}

	if !file.Exists(recommendListFile) {
		logger.Info("recommend list file not found", zap.String("recommendListFile", recommendListFile))
		return []string{}
	}

	buf := file.ReadFullFile(recommendListFile)
	if err := json.Unmarshal(buf, &recommendList); err != nil {
		logger.Error("failed to unmarshal recommend list", zap.Error(err), zap.String("recommendListFile", recommendListFile))
		return []string{}
	}

	result := lo.Map(recommendList, func(item interface{}, i int) string {
		recommendItem, ok := item.(map[string]interface{})
		if !ok {
			return ""
		}

		storeAppID, ok := recommendItem["appid"]
		if !ok {
			return ""
		}

		return storeAppID.(string)
	})

	return result
}

func BuildCatalog(storeRoot string) (map[string]*ComposeApp, error) {
	catalog := map[string]*ComposeApp{}

	// walk through each folder under storeRoot/Apps and build the catalog
	if err := filepath.WalkDir(filepath.Join(storeRoot, common.AppsDirectoryName), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		composeFile := filepath.Join(path, common.ComposeYAMLFileName)
		if !file.Exists(composeFile) {
			// retry with ".yaml" extension
			composeFile = strings.TrimSuffix(composeFile, ".yml") + ".yaml"
			if !file.Exists(composeFile) {
				return nil
			}
		}

		composeYAML := file.ReadFullFile(composeFile)
		if len(composeYAML) == 0 {
			return nil
		}

		composeApp, err := NewComposeAppFromYAML(composeYAML, true, false)
		if err != nil {
			logger.Info("failed to parse compose app - contact the contributor of this app to fix it", zap.Error(err), zap.String("composeFile", composeFile))
			return fs.SkipDir // skip invalid compose app
		}

		catalog[composeApp.Name] = composeApp

		return nil
	}); err != nil {
		return nil, err
	}

	return catalog, nil
}

func StoreRoot(workdir string) (string, error) {
	storeRoot := ""

	// locate the path that contains the Apps directory
	if err := filepath.WalkDir(workdir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && d.Name() == common.AppsDirectoryName {
			storeRoot = filepath.Dir(path)
			return filepath.SkipDir
		}

		return nil
	}); err != nil {
		return "", err
	}

	if storeRoot != "" {
		return storeRoot, nil
	}

	return "", ErrNotAppStore
}
