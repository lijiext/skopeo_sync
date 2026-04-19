package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/skopeo-sync/web/db"
	"github.com/skopeo-sync/web/handlers"
	"github.com/skopeo-sync/web/middleware"
	"github.com/skopeo-sync/web/scheduler"
)

// 简单的 CORS 中间件，允许前端调用
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func main() {
	// 初始化数据库
	err := db.InitDB()
	if err != nil {
		panic("数据库初始化失败: " + err.Error())
	}

	// 启动后台任务调度器
	scheduler.Start()

	r := gin.Default()
	
	// 使用跨域中间件
	r.Use(corsMiddleware())

	// 健康检查接口
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/api/captcha", handlers.GenerateCaptcha)

	// 登录接口 (公开)
	r.POST("/api/login", handlers.Login)

	// 提供前端静态文件服务，映射 / 路径到前端打包产物
	// 此处需要注意由于 vue router / history 模式的影响，如果刷新 404，可以使用 NoRoute 兜底返回 index.html
	r.Static("/assets", "./public/assets")
	r.StaticFile("/vite.svg", "./public/vite.svg")
	r.StaticFile("/", "./public/index.html")
	r.NoRoute(func(c *gin.Context) {
		c.File("./public/index.html")
	})

	// API 路由组 (需要鉴权)
	api := r.Group("/api")
	
	// WebSocket 动态日志接口 (无需走 HTTP token 鉴权拦截，或者带在 query)
	api.GET("/ws/logs", handlers.WsLogHandler)

	api.Use(middleware.AuthRequired())
	{
		// 仓库管理
		api.GET("/registries", handlers.GetRegistries)
		api.POST("/registries", handlers.CreateRegistry)
		api.PUT("/registries/:id", handlers.UpdateRegistry)
		api.DELETE("/registries/:id", handlers.DeleteRegistry)
		
		// 任务管理
		api.GET("/tasks", handlers.GetTasks)
		api.POST("/tasks", handlers.CreateTask)
		api.POST("/tasks/:id/retry", handlers.RetryTask)
		api.DELETE("/tasks/:id", handlers.DeleteTask)
		api.GET("/tasks/:id/logs", handlers.GetTaskLogs)

		// 配置管理 (如 webhook_url)
		api.GET("/config", handlers.GetConfig)
		api.POST("/config", handlers.SaveConfig)
	}

	// 启动服务器
	r.Run(":8080")
}
