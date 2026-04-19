package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/skopeo-sync/web/db"
	"github.com/skopeo-sync/web/models"
)

// GetConfig 获取指定的全局配置
func GetConfig(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "需要 key 参数"})
		return
	}

	var config models.SysConfig
	err := db.DB.Where("key = ?", key).First(&config).Error
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"key": key, "value": ""})
		return
	}
	c.JSON(http.StatusOK, config)
}

// SaveConfig 保存或更新全局配置
func SaveConfig(c *gin.Context) {
	var config models.SysConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if config.Key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "配置 Key 不能为空"})
		return
	}

	var exist models.SysConfig
	if err := db.DB.Where("key = ?", config.Key).First(&exist).Error; err == nil {
		// 已存在，执行更新
		db.DB.Model(&exist).Update("value", config.Value)
	} else {
		// 不存在，执行创建
		db.DB.Create(&config)
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
