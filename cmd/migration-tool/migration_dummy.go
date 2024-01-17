package main

import (
	"io"
	"os"
	"strings"

	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	interfaces "github.com/IceWhaleTech/CasaOS-Common"
)

const (
	oldAppStoreURL = "https://github.com/IceWhaleTech/_appstore/archive/refs/heads/main.zip"
	newAppStoreURL = "https://casaos-appstore.github.io/casaos-appstore/linux-all-appstore.zip"
)

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
	if strings.Contains(string(content), oldAppStoreURL) {
		return true, nil
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
	newContent := strings.Replace(string(content), oldAppStoreURL, newAppStoreURL, -1)
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
