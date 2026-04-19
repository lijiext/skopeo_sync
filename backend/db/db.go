package db

import (
	"github.com/skopeo-sync/web/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库连接并自动迁移表结构
func InitDB() error {
	var err error
	// 使用 skopeo.db 作为 SQLite 数据库文件
	DB, err = gorm.Open(sqlite.Open("skopeo.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // 生产环境关闭 SQL 日志
	})
	if err != nil {
		return err
	}
	
	// 自动迁移模型结构
	return DB.AutoMigrate(&models.Registry{}, &models.SyncTask{}, &models.SysConfig{})
}
