package main

import (
	"os"
	"time"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	interfaces "github.com/IceWhaleTech/CasaOS-Common"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/version"
	"gopkg.in/ini.v1"
)

type migrationTool038AndOlder struct{}

const (
	defaultConfigPath033to038    = "/etc/casaos.conf"
	defaultConfigPath032andOlder = "/casaOS/server/conf/conf.ini"
)

var defaultConfigPath string

func (u *migrationTool038AndOlder) IsMigrationNeeded() (bool, error) {
	_logger.Info("Checking if migration is needed...")

	if status, err := version.GetGlobalMigrationStatus(appManagementNameShort); err == nil {
		_status = status
		if status.LastMigratedVersion != "" {
			_logger.Info("Last migrated version: %s", status.LastMigratedVersion)
			if r, err := version.Compare(status.LastMigratedVersion, common.AppManagementVersion); err == nil {
				return r < 0, nil
			}
		}
	}

	if _, err := os.Stat(defaultConfigPath033to038); err != nil {
		if _, err := os.Stat(defaultConfigPath032andOlder); err != nil {
			_logger.Info("No legacy configuration found.")
			return false, nil
		} else {
			_logger.Info("`%s` found", defaultConfigPath032andOlder)
			defaultConfigPath = defaultConfigPath032andOlder
		}
	} else {
		_logger.Info("`%s` found", defaultConfigPath033to038)
		defaultConfigPath = defaultConfigPath033to038
	}

	var majorVersion, minorVersion, patchVersion int
	majorVersion, minorVersion, patchVersion, err := version.DetectVersion()
	if err != nil {
		_logger.Info("version not detected - trying to detect if it is a legacy version (v0.3.4 or earlier)...")
		majorVersion, minorVersion, patchVersion, err = version.DetectLegacyVersion()
		if err != nil {
			if err == version.ErrLegacyVersionNotFound {
				_logger.Info("legacy version not detected, migration is not needed.")
				return false, nil
			}

			_logger.Error("failed to detect legacy version: %s", err)
			return false, err
		}
	}

	_logger.Info("Detected version: %d.%d.%d", majorVersion, minorVersion, patchVersion)

	if majorVersion != 0 {
		return false, nil
	}

	if minorVersion > 3 {
		return false, nil
	}

	if minorVersion == 3 && patchVersion > 8 {
		return false, nil
	}

	_logger.Info("Migration is needed for a CasaOS version 0.3.8 and older...")
	return true, nil
}

func (u *migrationTool038AndOlder) PreMigrate() error {
	if _, err := os.Stat(appManagementConfigDirPath); os.IsNotExist(err) {
		_logger.Info("Creating %s since it doesn't exists...", appManagementConfigDirPath)
		if err := os.Mkdir(appManagementConfigDirPath, 0o755); err != nil {
			return err
		}
	}

	if _, err := os.Stat(appManagementConfigFilePath); os.IsNotExist(err) {
		_logger.Info("Creating %s since it doesn't exist...", appManagementConfigFilePath)

		f, err := os.Create(appManagementConfigFilePath)
		if err != nil {
			return err
		}

		defer f.Close()

		if _, err := f.WriteString(_appManagementConfigFileSample); err != nil {
			return err
		}
	}

	extension := "." + time.Now().Format("20060102") + ".bak"

	_logger.Info("Creating a backup %s if it doesn't exist...", defaultConfigPath+extension)
	return file.CopySingleFile(defaultConfigPath, defaultConfigPath+extension, "skip")
}

func (u *migrationTool038AndOlder) Migrate() error {
	_logger.Info("Loading legacy %s...", defaultConfigPath)
	legacyConfigFile, err := ini.Load(defaultConfigPath)
	if err != nil {
		return err
	}

	migrateConfigurationFile(legacyConfigFile)

	return nil
}

func (u *migrationTool038AndOlder) PostMigrate() error {
	defer func() {
		if err := _status.Done(common.AppManagementVersion); err != nil {
			_logger.Error("Failed to update migration status")
			panic(err)
		}
	}()

	return nil
}

func NewMigrationToolFor038AndOlder() interfaces.MigrationTool {
	return &migrationTool038AndOlder{}
}

func migrateConfigurationFile(legacyConfigFile *ini.File) {
	_logger.Info("Updating %s with settings from legacy configuration...", config.AppManagementConfigFilePath)
	config.InitSetup(config.AppManagementConfigFilePath)

	// LogPath
	if logPath, err := legacyConfigFile.Section("app").GetKey("LogPath"); err == nil {
		_logger.Info("[app] LogPath = %s", logPath.Value())
		config.AppInfo.LogPath = logPath.Value()
	}

	// LogFileExt
	if logFileExt, err := legacyConfigFile.Section("app").GetKey("LogFileExt"); err == nil {
		_logger.Info("[app] LogFileExt = %s", logFileExt.Value())
		config.AppInfo.LogFileExt = logFileExt.Value()
	}

	_logger.Info("Saving %s...", config.AppManagementConfigFilePath)
	config.SaveSetup(config.AppManagementConfigFilePath)
}
