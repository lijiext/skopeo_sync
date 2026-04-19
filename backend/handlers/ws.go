package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/skopeo-sync/web/db"
	"github.com/skopeo-sync/web/models"
)

var upgrader = websocket.Upgrader{
	// 允许跨域
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WsLogHandler WebSocket 动态推送日志接口
func WsLogHandler(c *gin.Context) {
	taskID := c.Query("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺失 task_id"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("升级 WebSocket 失败:", err)
		return
	}
	defer conn.Close()

	lastLen := 0
	// 定时轮询数据库（每秒查一次），发现长度变长则将全量推给前端，前端覆盖即可
	for {
		var task models.SyncTask
		if err := db.DB.First(&task, taskID).Error; err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte("错误: 任务不存在"))
			break
		}

		currentLog := task.Logs
		if len(currentLog) > lastLen {
			err = conn.WriteMessage(websocket.TextMessage, []byte(currentLog))
			if err != nil {
				log.Println("推送日志失败:", err)
				break
			}
			lastLen = len(currentLog)
		}

		// 检查任务是否已经彻底结束
		if task.Status == "success" || task.Status == "failed" {
			// 最后确认一下是否还有漏推的
			if len(task.Logs) > lastLen {
				conn.WriteMessage(websocket.TextMessage, []byte(task.Logs))
			}
			break
		}

		// 避免频繁查询
		time.Sleep(1 * time.Second)
	}
}
