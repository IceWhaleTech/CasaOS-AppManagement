package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"

	"github.com/IceWhaleTech/CasaOS-AppManagement/common"
	interfaces "github.com/IceWhaleTech/CasaOS-Common"
	"github.com/IceWhaleTech/CasaOS-Common/utils/systemctl"
)

const (
	appManagementConfigDirPath  = "/etc/casaos"
	appManagementConfigFilePath = "/etc/casaos/app-management.conf"
	appManagementName           = "casaos-app-management.service"
	appManagementNameShort      = "app-management"
)

//go:embedded ../../build/sysroot/etc/casaos/app-management.conf.sample
//var _appManagementConfigFileSample string

var (
	_logger *Logger
	// _status *version.GlobalMigrationStatus

	commit = "private build"
	date   = "private build"
)

func main() {
	versionFlag := flag.Bool("v", false, "version")
	debugFlag := flag.Bool("d", true, "debug")
	forceFlag := flag.Bool("f", false, "force")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("v%s\n", common.AppManagementVersion)
		os.Exit(0)
	}

	println("git commit:", commit)
	println("build date:", date)

	_logger = NewLogger()

	if os.Getuid() != 0 {
		_logger.Info("Root privileges are required to run this program.")
		os.Exit(1)
	}

	if *debugFlag {
		_logger.DebugMode = true
	}

	if !*forceFlag {
		isRunning, err := systemctl.IsServiceRunning(appManagementName)
		if err != nil {
			_logger.Error("Failed to check if %s is running", appManagementName)
			panic(err)
		}

		if isRunning {
			_logger.Info("%s is running. If migration is still needed, try with -f.", appManagementName)
			os.Exit(1)
		}
	}

	migrationTools := []interfaces.MigrationTool{
		// NewMigrationDummy(),
		NewMigration044AndOlder(),
		NewMigration0412AndOlder(),
	}

	var selectedMigrationTool interfaces.MigrationTool

	// look for the right migration tool matching current version
	for _, tool := range migrationTools {
		migrationNeeded, _ := tool.IsMigrationNeeded()

		if migrationNeeded {
			selectedMigrationTool = tool
			break
		}
	}

	if selectedMigrationTool == nil {
		_logger.Info("No migration to proceed.")
		return
	}

	if err := selectedMigrationTool.PreMigrate(); err != nil {
		panic(err)
	}

	if err := selectedMigrationTool.Migrate(); err != nil {
		panic(err)
	}

	if err := selectedMigrationTool.PostMigrate(); err != nil {
		_logger.Error("Migration succeeded, but post-migration failed: %s", err)
	}
}
