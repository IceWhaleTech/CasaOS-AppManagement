package service

import (
	"crypto/md5" // nolint: gosec
	_ "embed"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/utils/downloadHelper"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"go.uber.org/zap"
)

type AppStore struct {
	url     string
	catalog map[string]*ComposeApp
}

var (
	//go:embed fixtures/sample.docker-compose.yaml
	SampleComposeAppYAML string

	ErrNotAppStore = fmt.Errorf("not an appstore")
)

func (s *AppStore) UpdateCatalog() error {
	if _, err := url.Parse(s.url); err != nil {
		return err
	}

	workdir, err := s.WorkDir()
	if err != nil {
		return err
	}

	updateSucessful := false

	if file.Exists(workdir) {
		backupDir := workdir + ".backup"
		if err := os.Rename(workdir, backupDir); err != nil {
			return err
		}

		defer func() {
			if !updateSucessful {
				if err := file.RMDir(workdir); err != nil {
					logger.Error("failed to remove appstore workdir", zap.Error(err), zap.String("workdir", workdir))
				}

				if err := os.Rename(backupDir, workdir); err != nil {
					logger.Error("failed to restore appstore workdir", zap.Error(err), zap.String("backupDir", backupDir), zap.String("workdir", workdir))
				}
			}
		}()
	}

	if err := prepare(workdir, s.url); err != nil {
		return err
	}

	if err := downloadHelper.Download(s.url, workdir); err != nil {
		return err
	}

	// TODO - implement this

	updateSucessful = true

	return nil
}

func (s *AppStore) Catalog() map[string]*ComposeApp {
	return s.catalog
}

func (s *AppStore) ComposeApp(appStoreID string) *ComposeApp {
	if composeApp, ok := s.catalog[appStoreID]; ok {
		return composeApp
	}

	return nil
}

func (s *AppStore) WorkDir() (string, error) {
	parsedURL, err := url.Parse(s.url)
	if err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", md5.Sum([]byte(parsedURL.Path))) //nolint: gosec

	return filepath.Join(config.AppInfo.AppStorePath, parsedURL.Host, hash), nil
}

func NewAppStore(appstoreURL string) (*AppStore, error) {
	appstoreURL = strings.ToLower(appstoreURL)

	_, err := url.Parse(appstoreURL)
	if err != nil {
		return nil, err
	}

	return &AppStore{
		url:     appstoreURL,
		catalog: map[string]*ComposeApp{},
	}, nil
}

func prepare(workdir string, markerContent string) error {
	if err := file.IsNotExistMkDir(workdir); err != nil {
		return err
	}

	placeholderFile := filepath.Join(workdir, ".casaos-appstore")
	return file.CreateFileAndWriteContent(placeholderFile, markerContent)
}

func storeRoot(workdir string) (string, error) {
	storeRoot := ""

	// locate the path that contains the Apps directory
	if err := filepath.WalkDir(workdir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && d.Name() == "Apps" {
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
