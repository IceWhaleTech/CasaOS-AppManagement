package sqlite

import (
	"time"

	model2 "github.com/IceWhaleTech/CasaOS-AppManagement/service/model"
	"github.com/IceWhaleTech/CasaOS-Common/utils/file"
	"github.com/IceWhaleTech/CasaOS-Common/utils/logger"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var gdb *gorm.DB

func GetGlobalDB(dbPath string) *gorm.DB {
	if gdb != nil {
		return gdb
	}

	if err := file.IsNotExistMkDir(dbPath); err != nil {
		logger.Error("error when creating database directory", zap.Any("error", err), zap.String("path", dbPath))
		return nil
	}

	db, err := gorm.Open(sqlite.Open(dbPath+"/app-management.db"), &gorm.Config{})
	if err != nil {
		logger.Error("sqlite connect error", zap.Any("db connect error", err))
		return nil
	}

	gdb = db

	c, _ := gdb.DB()
	c.SetMaxIdleConns(10)
	c.SetMaxOpenConns(100)
	c.SetConnMaxIdleTime(time.Second * 1000)

	if err := gdb.AutoMigrate(&model2.AppListDBModel{}); err != nil {
		logger.Error("check or create db error", zap.Any("error", err))
	}
	return gdb
}
