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
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"go.uber.org/zap"
)

type AppStore struct {
	url     string
	catalog map[string]*ComposeApp
}

//go:embed fixtures/sample.docker-compose.yaml
var SampleComposeAppYAML string

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
				if err := os.Rename(backupDir, workdir); err != nil {
					logger.Error("failed to restore appstore workdir", zap.Error(err), zap.String("backupDir", backupDir), zap.String("workdir", workdir))
				}
			}
		}()
	}

	if err := s.prepareWorkDir(); err != nil {
		return err
	}

	// TODO - download .zip file from s.url using github.com/hashicorp/go-getter

	// unzip it to workdir

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

func (s *AppStore) prepareWorkDir() error {
	workdir, err := s.WorkDir()
	if err != nil {
		return err
	}

	if err := file.IsNotExistMkDir(workdir); err != nil {
		return err
	}

	placeholderFile := filepath.Join(workdir, ".casaos-appstore")
	return file.CreateFileAndWriteContent(placeholderFile, s.url)
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
