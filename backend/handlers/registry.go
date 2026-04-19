package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/skopeo-sync/web/db"
	"github.com/skopeo-sync/web/models"
)

// GetRegistries 获取所有仓库配置
func GetRegistries(c *gin.Context) {
	var registries []models.Registry
	db.DB.Find(&registries)
	c.JSON(http.StatusOK, registries)
}

// CreateRegistry 创建新仓库配置
func CreateRegistry(c *gin.Context) {
	var registry models.Registry
	if err := c.ShouldBindJSON(&registry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	db.DB.Create(&registry)
	c.JSON(http.StatusOK, registry)
}

// UpdateRegistry 更新仓库配置
func UpdateRegistry(c *gin.Context) {
	id := c.Param("id")
	var registry models.Registry
	if err := db.DB.First(&registry, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该仓库"})
		return
	}

	var req models.Registry
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	registry.Name = req.Name
	registry.URL = req.URL
	registry.Username = req.Username
	// 如果传了密码，才更新密码
	if req.Password != "" {
		registry.Password = req.Password
	}

	db.DB.Save(&registry)
	c.JSON(http.StatusOK, registry)
}

// DeleteRegistry 删除仓库配置
func DeleteRegistry(c *gin.Context) {
	id := c.Param("id")
	var registry models.Registry
	if err := db.DB.First(&registry, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该仓库"})
		return
	}
	db.DB.Delete(&registry)
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
