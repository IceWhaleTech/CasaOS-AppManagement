package main

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	"github.com/IceWhaleTech/CasaOS-AppManagement/pkg/config"
	interfaces "github.com/IceWhaleTech/CasaOS-Common"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/version"
	"gopkg.in/ini.v1"
)

type migrationTool1 struct{}

const (
	defaultDBPath = "/var/lib/casaos"
	tableName     = "o_container"
)

func (u *migrationTool1) IsMigrationNeeded() (bool, error) {
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

	if _, err = os.Stat(version.LegacyCasaOSConfigFilePath); err != nil {
		_logger.Info("`%s` not found, migration is not needed.", version.LegacyCasaOSConfigFilePath)
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

	if minorVersion < 2 {
		return false, nil
	}

	if minorVersion == 3 && patchVersion > 8 {
		return false, nil
	}

	legacyConfigFile, err := ini.Load(version.LegacyCasaOSConfigFilePath)
	if err != nil {
		_logger.Error("failed to load config file %s - %s", version.LegacyCasaOSConfigFilePath, err.Error())
		return false, err
	}

	dbFile, err := getDBfile(legacyConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			_logger.Info("database file not found from %s, migration is not needed.", version.LegacyCasaOSConfigFilePath)
			return false, nil
		}

		_logger.Error("failed to get database file from %s - %s", version.LegacyCasaOSConfigFilePath, err.Error())
		return false, err
	}

	legacyDB, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		_logger.Error("failed to open database file %s - %s", dbFile, err.Error())
		return false, err
	}

	defer legacyDB.Close()

	tableExists, err := isTableExist(legacyDB, tableName)
	if err != nil {
		_logger.Error("failed to check if table %s exists - %s", tableName, err.Error())
		return false, err
	}

	if !tableExists {
		_logger.Info("table %s does not exist, migration is not needed.", tableName)
		return false, nil
	}

	_logger.Info("Migration is needed for a CasaOS version between 0.2.x and 0.3.8...")
	return true, nil
}

func (u *migrationTool1) PreMigrate() error {
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

	_logger.Info("Creating a backup %s if it doesn't exist...", version.LegacyCasaOSConfigFilePath+extension)
	if err := file.CopySingleFile(version.LegacyCasaOSConfigFilePath, version.LegacyCasaOSConfigFilePath+extension, "skip"); err != nil {
		return err
	}

	legacyConfigFile, err := ini.Load(version.LegacyCasaOSConfigFilePath)
	if err != nil {
		return err
	}

	dbFile, err := getDBfile(legacyConfigFile)
	if err != nil {
		return err
	}

	_logger.Info("Creating a backup %s if it doesn't exist...", dbFile+extension)
	if err := file.CopySingleFile(dbFile, dbFile+extension, "skip"); err != nil {
		return err
	}

	return nil
}

func (u *migrationTool1) Migrate() error {
	_logger.Info("Loading legacy %s...", version.LegacyCasaOSConfigFilePath)
	legacyConfigFile, err := ini.Load(version.LegacyCasaOSConfigFilePath)
	if err != nil {
		return err
	}

	migrateConfigurationFile1(legacyConfigFile)

	return nil
}

func (u *migrationTool1) PostMigrate() error {
	defer func() {
		if err := _status.Done(common.AppManagementVersion); err != nil {
			_logger.Error("Failed to update migration status")
			panic(err)
		}
	}()

	legacyConfigFile, err := ini.Load(version.LegacyCasaOSConfigFilePath)
	if err != nil {
		return err
	}

	return postMigrateConfigurationFile1(legacyConfigFile)
}

func NewMigrationToolFor038AndOlder() interfaces.MigrationTool {
	return &migrationTool1{}
}

func migrateConfigurationFile1(legacyConfigFile *ini.File) {
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

	// DBPath
	if dbPath, err := legacyConfigFile.Section("app").GetKey("DBPath"); err == nil {
		_logger.Info("[app] DBPath = %s", dbPath.Value())
		config.AppInfo.DBPath = dbPath.Value() + "/db"
	}

	_logger.Info("Saving %s...", config.AppManagementConfigFilePath)
	config.SaveSetup(config.AppManagementConfigFilePath)
}

func postMigrateConfigurationFile1(legacyConfigFile *ini.File) error {
	// do nothing
	return nil
}

func isTableExist(legacyDB *sql.DB, tableName string) (bool, error) {
	rows, err := legacyDB.Query("SELECT name FROM sqlite_master WHERE type='table' AND name = ?", tableName)
	if err != nil {
		return false, err
	}

	defer rows.Close()

	return rows.Next(), nil
}

func getDBfile(legacyConfigFile *ini.File) (string, error) {
	if legacyConfigFile == nil {
		return "", errors.New("legacy configuration file is nil")
	}

	dbPath := legacyConfigFile.Section("app").Key("DBPath").String()

	dbFile := filepath.Join(dbPath, "db", "casaOS.db")

	if _, err := os.Stat(dbFile); err != nil {
		dbFile = filepath.Join(defaultDBPath, "db", "casaOS.db")

		if _, err := os.Stat(dbFile); err != nil {
			return "", err
		}
	}

	return dbFile, nil
}
