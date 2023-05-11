package service

import (
	"crypto/md5" // nolint: gosec
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

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
	CategoryMap() map[string]codegen.CategoryInfo
	Recommend() []string

	Catalog() map[string]*ComposeApp
	ComposeApp(id string) *ComposeApp
	UpdateCatalog() error
	WorkDir() (string, error)
}

type appStore struct {
	catalog map[string]*ComposeApp
	url     string
}

var (
	appStoreMap = make(map[string]*appStore)

	ErrNotAppStore             = fmt.Errorf("not an appstore")
	ErrDefaultAppStoreNotFound = fmt.Errorf("default appstore not found")
)

func (s *appStore) CategoryMap() map[string]codegen.CategoryInfo {
	workdir, err := s.WorkDir()
	if err != nil {
		return nil
	}

	storeRoot, err := StoreRoot(workdir)
	if err != nil {
		return nil
	}

	return LoadCategoryMap(storeRoot)
}

func (s *appStore) UpdateCatalog() error {
	if _, err := url.Parse(s.url); err != nil {
		return err
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

	isSuccessful := false

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

	isSuccessful = true

	return nil
}

func (s *appStore) Recommend() []string {
	workdir, err := s.WorkDir()
	if err != nil {
		return nil
	}

	storeRoot, err := StoreRoot(workdir)
	if err != nil {
		return nil
	}

	return LoadRecommend(storeRoot)
}

func (s *appStore) Catalog() map[string]*ComposeApp {
	if s.catalog != nil && len(s.catalog) > 0 {
		return s.catalog
	}

	workdir, err := s.WorkDir()
	if err != nil {
		return nil
	}

	storeRoot, err := StoreRoot(workdir)
	if err != nil {
		return nil
	}

	catalog, err := BuildCatalog(storeRoot)
	if err != nil {
		return nil
	}

	s.catalog = catalog

	return s.catalog
}

func (s *appStore) ComposeApp(appStoreID string) *ComposeApp {
	catalog := s.Catalog()

	if catalog == nil {
		return nil
	}

	if composeApp, ok := catalog[appStoreID]; ok {
		return composeApp
	}

	return nil
}

func (s *appStore) WorkDir() (string, error) {
	parsedURL, err := url.Parse(s.url)
	if err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", md5.Sum([]byte(parsedURL.Path))) //nolint: gosec

	return filepath.Join(config.AppInfo.AppStorePath, parsedURL.Host, hash), nil
}

func AppStoreByURL(appstoreURL string) (AppStore, error) {
	appstoreURL = strings.ToLower(appstoreURL)

	_, err := url.Parse(appstoreURL)
	if err != nil {
		return nil, err
	}

	if appstore, ok := appStoreMap[appstoreURL]; ok {
		return appstore, nil
	}

	appStoreMap[appstoreURL] = &appStore{
		url:     appstoreURL,
		catalog: map[string]*ComposeApp{},
	}

	return appStoreMap[appstoreURL], nil
}

func NewDefaultAppStore() (AppStore, error) {
	storeRoot := filepath.Join(config.AppInfo.AppStorePath, "default")

	if !file.Exists(storeRoot) {
		return nil, ErrDefaultAppStoreNotFound
	}

	catalog, err := BuildCatalog(storeRoot)
	if err != nil {
		return nil, err
	}

	return &appStore{
		url:     "default",
		catalog: catalog,
	}, nil
}

func LoadCategoryMap(storeRoot string) map[string]codegen.CategoryInfo {
	categoryListFile := filepath.Join(storeRoot, common.CategoryListFileName)

	// unmarsal category list
	categoryList := []codegen.CategoryInfo{}

	if file.Exists(categoryListFile) {
		buf := file.ReadFullFile(categoryListFile)

		if err := json.Unmarshal(buf, &categoryList); err != nil {
			logger.Error("failed to unmarshal category list", zap.Error(err), zap.String("categoryListFile", categoryListFile))
		}
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

	if file.Exists(recommendListFile) {
		buf := file.ReadFullFile(recommendListFile)

		if err := json.Unmarshal(buf, &recommendList); err != nil {
			logger.Error("failed to unmarshal recommend list", zap.Error(err), zap.String("recommendListFile", recommendListFile))
		}
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
			return nil
		}

		composeYAML := file.ReadFullFile(composeFile)
		if len(composeYAML) == 0 {
			return nil
		}

		composeApp, err := NewComposeAppFromYAML(composeYAML, true, false)
		if err != nil {
			return err
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
