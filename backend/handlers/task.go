package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/skopeo-sync/web/db"
	"github.com/skopeo-sync/web/models"
)

// GetTasks 获取同步任务列表
func GetTasks(c *gin.Context) {
	var tasks []models.SyncTask
	db.DB.Order("id desc").Limit(100).Find(&tasks)
	c.JSON(http.StatusOK, tasks)
}

// DeleteTask 删除同步任务
func DeleteTask(c *gin.Context) {
	id := c.Param("id")
	var task models.SyncTask
	if err := db.DB.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该任务"})
		return
	}

	// 不允许删除正在运行的任务
	if task.Status == "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除正在运行中的任务"})
		return
	}

	db.DB.Delete(&task)
	c.JSON(http.StatusOK, gin.H{"message": "任务删除成功"})
}

// 批量创建任务请求体
type CreateTaskReq struct {
	SourceID uint   `json:"source_id"`
	DestID   uint   `json:"dest_id"`
	Images   string `json:"images"` // 按换行符分隔的多个镜像
	Retries  int    `json:"retries"`
}

// CreateTask 创建新同步任务 (支持批量创建)
func CreateTask(c *gin.Context) {
	var req CreateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	if req.SourceID == req.DestID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "源仓库和目标仓库不能为同一个"})
		return
	}

	// 先查询出目标仓库，用于智能推断 destImage
	var dest models.Registry
	db.DB.First(&dest, req.DestID)

	// 拆分多行镜像
	imageList := strings.Split(req.Images, "\n")
	var createdTasks []models.SyncTask

	for _, line := range imageList {
		line = strings.TrimSpace(line)
		if line == "" {
			continue // 跳过空行
		}

		var srcImg, destImg string
		// 判断是否使用了 `->` 语法显式指定了源和目标
		if strings.Contains(line, "->") {
			parts := strings.Split(line, "->")
			if len(parts) == 2 {
				srcImg = strings.TrimSpace(parts[0])
				destImg = strings.TrimSpace(parts[1])
			}
		} else {
			srcImg = line
			
			// 智能推断目标名称
			// 如果目标库配置了用户名（比如 dockerhub 的 openclover），就把前缀替换掉
			// 比如: library/nginx:latest -> openclover/nginx:latest
			shortName := srcImg
			if idx := strings.LastIndex(srcImg, "/"); idx != -1 {
				shortName = srcImg[idx+1:]
			}
			
			if dest.Username != "" {
				destImg = dest.Username + "/" + shortName
			} else {
				destImg = srcImg
			}
		}

		task := models.SyncTask{
			SourceID:  req.SourceID,
			DestID:    req.DestID,
			Image:     srcImg,
			DestImage: destImg,
			Status:    "pending",
			Retries:   req.Retries,
		}

		if task.Retries <= 0 {
			task.Retries = 3
		}

		db.DB.Create(&task)
		createdTasks = append(createdTasks, task)
	}

	c.JSON(http.StatusOK, createdTasks)
}

// RetryTask 手动触发任务重试
func RetryTask(c *gin.Context) {
	taskID := c.Param("id")
	var task models.SyncTask
	
	if err := db.DB.First(&task, taskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	// 状态重置为 pending，让调度器重新拾取
	db.DB.Model(&task).Updates(map[string]interface{}{
		"status":        "pending",
		"current_retry": 0,
		"logs":          "--- 手动触发重试 ---\n",
	})

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "任务已重置为等待状态"})
}

// GetTaskLogs 获取静态日志
func GetTaskLogs(c *gin.Context) {
	taskID := c.Param("id")
	var task models.SyncTask
	
	if err := db.DB.First(&task, taskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"logs": task.Logs})
}
