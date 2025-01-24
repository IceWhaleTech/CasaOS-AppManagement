package main

import (
	"io"
	"os"
	"strings"

	interfaces "github.com/IceWhaleTech/CasaOS-Common"

	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
)

type UrlReplacement struct {
	OldUrl string
	NewUrl string
}

var replaceUrl = []UrlReplacement{
	{
		OldUrl: "https://github.com/IceWhaleTech/_appstore/archive/refs/heads/main.zip",
		NewUrl: "https://casaos.app/store/main.zip",
	},
	{
		OldUrl: "https://casaos.oss-cn-shanghai.aliyuncs.com/IceWhaleTech/_appstore/archive/refs/heads/main.zip",
		NewUrl: "https://cdn.jsdelivr.net/gh/IceWhaleTech/CasaOS-AppStore@gh-pages/store/main.zip",
	},
}

type migrationTool0415AndOlder struct{}

func (u *migrationTool0415AndOlder) IsMigrationNeeded() (bool, error) {
	_logger.Info("Checking if migration is needed...")

	// read string from AppManagementConfigFilePath
	file, err := os.Open(config.AppManagementConfigFilePath)
	if err != nil {
		_logger.Error("failed to detect app management config file: %s", err)
		return false, err
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		_logger.Error("failed to read app management config file: %s", err)
		return false, err
	}

	for _, v := range replaceUrl {
		if strings.Contains(string(content), v.OldUrl) {
			_logger.Info("Migration is needed for a CasaOS with old app store link.")
			return true, nil
		}
	}
	return false, nil
}

func (u *migrationTool0415AndOlder) PreMigrate() error {
	return nil
}

func (u *migrationTool0415AndOlder) Migrate() error {
	// replace string in AppManagementConfigFilePath
	// replace https://github.com/IceWhaleTech/_appstore/archive/refs/heads/main.zip to https://casaos-appstore.github.io/casaos-appstore/linux-all-appstore.zip
	file, err := os.OpenFile(config.AppManagementConfigFilePath, os.O_RDWR, 0644)
	if err != nil {
		_logger.Error("failed to open app management config file: %s", err)
		return err
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		_logger.Error("failed to read app management config file: %s", err)
		return err
	}

	newContent := string(content)
	for _, v := range replaceUrl {
		newContent = strings.Replace(newContent, v.OldUrl, v.NewUrl, -1)
	}

	// clear the ole content
	err = file.Truncate(0)
	if err != nil {
		_logger.Error("failed to truncate app management config file: %s", err)
		return err
	}

	_, err = file.WriteAt([]byte(newContent), 0)
	if err != nil {
		_logger.Error("failed to write app management config file: %s", err)
		return err
	}
	return nil
}

func (u *migrationTool0415AndOlder) PostMigrate() error {
	return nil
}

func NewMigration0415AndOlder() interfaces.MigrationTool {
	return &migrationTool0415AndOlder{}
}
