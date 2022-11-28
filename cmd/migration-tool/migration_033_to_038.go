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

type migrationTool033to038 struct{}

const (
	defaultDBPath033to038     = "/var/lib/casaos"
	defaultConfigPath033to038 = "/etc/casaos.conf"
)

func (u *migrationTool033to038) IsMigrationNeeded() (bool, error) {
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

	if _, err = os.Stat(defaultConfigPath033to038); err != nil {
		_logger.Info("`%s` not found, migration is not needed.", defaultConfigPath033to038)
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

	if minorVersion != 3 {
		return false, nil
	}

	if patchVersion < 3 && patchVersion > 8 {
		return false, nil
	}

	_logger.Info("Migration is needed for a CasaOS version between 0.3.3 and 0.3.8...")
	return true, nil
}

func (u *migrationTool033to038) PreMigrate() error {
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

	_logger.Info("Creating a backup %s if it doesn't exist...", defaultConfigPath033to038+extension)
	return file.CopySingleFile(defaultConfigPath033to038, defaultConfigPath033to038+extension, "skip")
}

func (u *migrationTool033to038) Migrate() error {
	_logger.Info("Loading legacy %s...", defaultConfigPath033to038)
	legacyConfigFile, err := ini.Load(defaultConfigPath033to038)
	if err != nil {
		return err
	}

	migrateConfigurationFile033to038(legacyConfigFile)

	return nil
}

func (u *migrationTool033to038) PostMigrate() error {
	defer func() {
		if err := _status.Done(common.AppManagementVersion); err != nil {
			_logger.Error("Failed to update migration status")
			panic(err)
		}
	}()

	return nil
}

func NewMigrationToolFor033to038() interfaces.MigrationTool {
	return &migrationTool033to038{}
}

func migrateConfigurationFile033to038(legacyConfigFile *ini.File) {
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
