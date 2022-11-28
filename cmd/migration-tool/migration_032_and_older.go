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

type migrationTool032AndOlder struct{}

const (
	defaultDBPath032AndOlder     = "/casaOS/server"
	defaultConfigPath032AndOlder = "/casaOS/server/conf/conf.ini"
)

func (u *migrationTool032AndOlder) IsMigrationNeeded() (bool, error) {
	status, err := version.GetGlobalMigrationStatus(appManagementNameShort)
	if err != nil {
		_status = status
		if status.LastMigratedVersion != "" {
			_logger.Info("Last migrated version: %s", status.LastMigratedVersion)
			if r, err := version.Compare(status.LastMigratedVersion, common.AppManagementVersion); err != nil {
				return r < 0, nil
			}
		}
	}

	if _, err = os.Stat(defaultConfigPath032AndOlder); err != nil {
		_logger.Info("`%s` not found, migration is not needed.", defaultConfigPath032AndOlder)
		return false, err
	}

	var majorVersion, minorVersion, patchVersion int
	majorVersion, minorVersion, patchVersion, err = version.DetectVersion()
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

	if majorVersion != 0 {
		return false, nil
	}

	if minorVersion == 2 {
		_logger.Info("Migration is needed for a CasaOS version 0.2.x...")
		return true, nil
	}

	if minorVersion == 3 && patchVersion < 3 {
		_logger.Info("Migration is needed for a CasaOS version between 0.3.0 and 0.3.2...")
		return true, nil
	}

	return false, nil
}

func (u *migrationTool032AndOlder) PreMigrate() error {
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

	_logger.Info("Creating a backup %s if it doesn't exist...", defaultConfigPath032AndOlder+extension)
	return file.CopySingleFile(defaultConfigPath032AndOlder, defaultConfigPath032AndOlder+extension, "skip")
}

func (u *migrationTool032AndOlder) Migrate() error {
	_logger.Info("Loading legacy %s...", defaultConfigPath032AndOlder)
	legacyConfigFile, err := ini.Load(defaultConfigPath032AndOlder)
	if err != nil {
		return err
	}

	migrateConfigurationFile032AndOlder(legacyConfigFile)

	return nil
}

func (u *migrationTool032AndOlder) PostMigrate() error {
	defer func() {
		if err := _status.Done(common.AppManagementVersion); err != nil {
			_logger.Error("Failed to update migration status")
			panic(err)
		}
	}()

	return nil
}

func NewMigrationToolFor032AndOlder() interfaces.MigrationTool {
	return &migrationTool032AndOlder{}
}

func migrateConfigurationFile032AndOlder(legacyConfigFile *ini.File) {
	_logger.Info("Updating %s with settings from legacy configuration...", config.AppManagementConfigFilePath)
	config.InitSetup(config.AppManagementConfigFilePath)

	// LogPath
	if logPath, err := legacyConfigFile.Section("app").GetKey("LogSavePath"); err == nil {
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
