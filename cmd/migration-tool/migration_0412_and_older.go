package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	interfaces "github.com/IceWhaleTech/CasaOS-Common"

	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
)

type migrationTool0412AndOlder struct{}

const bigBearAppStoreUrl = "https://github.com/bigbeartechworld/big-bear-casaos/archive/refs/heads/master.zip"

func (u *migrationTool0412AndOlder) IsMigrationNeeded() (bool, error) {
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

	if strings.Contains(string(content), bigBearAppStoreUrl) {
		_logger.Info("Migration is add big bear app store. it is not needed.")
		return false, nil
	}
	return true, nil
}

func (u *migrationTool0412AndOlder) PreMigrate() error {
	return nil
}

func (u *migrationTool0412AndOlder) Migrate() error {
	// add big bear app store
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
	newContent += fmt.Sprintf("\nappstore = %s", bigBearAppStoreUrl)

	_, err = file.WriteString(newContent)
	if err != nil {
		_logger.Error("failed to write app management config file: %s", err)
		return err
	}

	return nil
}

func (u *migrationTool0412AndOlder) PostMigrate() error {
	return nil
}

func NewMigration0412AndOlder() interfaces.MigrationTool {
	return &migrationTool0412AndOlder{}
}
