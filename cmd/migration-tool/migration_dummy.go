package main

import (
	"io"
	"os"
	"strings"

	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	interfaces "github.com/IceWhaleTech/CasaOS-Common"
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
		OldUrl: " https://casaos.oss-cn-shanghai.aliyuncs.com/IceWhaleTech/_appstore/archive/refs/heads/main.zip",
		NewUrl: "https://casaos.oss-cn-shanghai.aliyuncs.com/store/main.zip",
	},
}

type migrationTool struct{}

func (u *migrationTool) IsMigrationNeeded() (bool, error) {
	// read string from AppManagementConfigFilePath
	file, err := os.Open(config.AppManagementConfigFilePath)
	if err != nil {
		return false, err
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return false, err
	}
	for _, v := range replaceUrl {
		if strings.Contains(string(content), v.OldUrl) {
			return true, nil
		}
	}

	return false, nil
}

func (u *migrationTool) PreMigrate() error {
	return nil
}

func (u *migrationTool) Migrate() error {
	// replace string in AppManagementConfigFilePath
	// replace https://github.com/IceWhaleTech/_appstore/archive/refs/heads/main.zip to https://casaos-appstore.github.io/casaos-appstore/linux-all-appstore.zip
	file, err := os.OpenFile(config.AppManagementConfigFilePath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	newContent := string(content)
	for _, v := range replaceUrl {
		newContent = strings.Replace(string(newContent), v.OldUrl, v.NewUrl, -1)
	}

	_, err = file.WriteAt([]byte(newContent), 0)
	if err != nil {
		return err
	}
	return nil
}

func (u *migrationTool) PostMigrate() error {
	return nil
}

func NewMigrationDummy() interfaces.MigrationTool {
	return &migrationTool{}
}
