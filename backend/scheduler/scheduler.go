package scheduler

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/skopeo-sync/web/db"
	"github.com/skopeo-sync/web/engine"
	"github.com/skopeo-sync/web/models"
)

// Start 启动后台任务调度器
func Start() {
	go func() {
		log.Println("任务调度器已启动，正在轮询...")
		for {
			var task models.SyncTask
			// 查找一个状态为 pending 的任务
			result := db.DB.Where("status = ?", "pending").First(&task)
			if result.Error == nil {
				// 处理该任务
				processTask(&task)
			}
			// 避免 CPU 空转，每 5 秒轮询一次
			time.Sleep(5 * time.Second)
		}
	}()
}

// processTask 实际执行任务的逻辑
func processTask(task *models.SyncTask) {
	// 将任务状态更新为运行中
	db.DB.Model(task).Update("status", "running")

	var src, dest models.Registry
	// 获取源和目标仓库的详细信息
	db.DB.First(&src, task.SourceID)
	db.DB.First(&dest, task.DestID)

	logMsg := fmt.Sprintf("========== 任务ID: %d ==========\n", task.ID)
	logMsg += fmt.Sprintf("源镜像: %s/%s\n", src.URL, task.Image)
	logMsg += fmt.Sprintf("目标镜像: %s/%s\n", dest.URL, task.DestImage)
	logMsg += fmt.Sprintf("最大重试次数: %d\n", task.Retries)
	logMsg += "----------------------------------------\n"
	log.Printf("[任务调度] 开始同步任务 ID: %d, %s/%s -> %s/%s\n", task.ID, src.URL, task.Image, dest.URL, task.DestImage)
	
	// 初始化到内存和数据库
	task.Logs += logMsg
	db.DB.Model(task).UpdateColumn("logs", task.Logs)

	var logMutex sync.Mutex
	var pendingLogs string

	logCallback := func(msg string) {
		logMutex.Lock()
		pendingLogs += msg
		task.Logs += msg
		logMutex.Unlock()
	}

	// 开启一个 goroutine 每 500ms 将 pendingLogs 刷入 DB
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-time.After(500 * time.Millisecond):
				logMutex.Lock()
				if pendingLogs != "" {
					db.DB.Model(task).UpdateColumn("logs", task.Logs)
					pendingLogs = ""
				}
				logMutex.Unlock()
			case <-done:
				return
			}
		}
	}()

	// 调用底层 Skopeo 引擎执行同步，带上仓库认证信息
	// 这里的 retries 传 1，让调度器控制外层重试循环，以捕获每一次的执行日志
	err := engine.SyncImage(
		src.URL, dest.URL, task.Image, task.DestImage, 1,
		src.Username, src.Password, dest.Username, dest.Password,
		logCallback,
	)

	close(done)
	// 确保最后一点日志刷入
	logMutex.Lock()
	if pendingLogs != "" {
		db.DB.Model(task).UpdateColumn("logs", task.Logs)
		pendingLogs = ""
	}
	logMutex.Unlock()

	if err != nil {
		failMsg := fmt.Sprintf("同步失败: %v\n", err)
		log.Printf("[任务调度] 任务 ID: %d 同步失败\n", task.ID)
		task.Logs += failMsg

		task.CurrentRetry++
		if task.CurrentRetry >= task.Retries {
			// 重试次数耗尽，彻底失败
			task.Status = "failed"
			db.DB.Model(task).Updates(map[string]interface{}{
				"status": "failed",
				"logs":   task.Logs,
				"current_retry": task.CurrentRetry,
			})
			engine.SendWebhook(task, "任务达到最大重试次数，彻底失败。")
		} else {
			// 重置为 pending 等待下次轮询重试
			task.Logs += fmt.Sprintf("--- 准备进行第 %d 次重试 ---\n", task.CurrentRetry+1)
			db.DB.Model(task).Updates(map[string]interface{}{
				"status": "pending",
				"logs":   task.Logs,
				"current_retry": task.CurrentRetry,
			})
		}
		return
	}

	// 同步完成后执行一致性核验
	verifyMsg := "同步成功，开始校验镜像一致性...\n"
	log.Printf("[任务调度] 同步完成，开始校验镜像一致性... 任务 ID: %d\n", task.ID)
	
	// 计算消耗流量（单位：字节），这里获取目标仓库的真实占用大小
	trafficBytes := engine.GetImageSize(dest.URL, task.DestImage, dest.Username, dest.Password)

	valid, verifyLog := engine.VerifyImage(
		src.URL, dest.URL, task.Image, task.DestImage,
		src.Username, src.Password, dest.Username, dest.Password,
	)

	verifyMsg += verifyLog

	if !valid {
		verifyMsg += "校验失败: 源和目标的 Digest 不匹配\n"
		log.Printf("[任务调度] 任务 ID: %d 校验失败\n", task.ID)
		task.Status = "failed"
		db.DB.Model(task).Updates(map[string]interface{}{
			"status": "failed",
			"logs":   task.Logs + verifyMsg,
		})
		engine.SendWebhook(task, "镜像同步完成，但哈希校验失败。")
		return
	}

	// 校验通过，标记为成功
	verifyMsg += "校验通过，同步成功！\n"
	log.Printf("[任务调度] 任务 ID: %d 校验通过，同步成功！\n", task.ID)

	task.Status = "success"
	db.DB.Model(task).Updates(map[string]interface{}{
		"status": "success",
		"traffic_bytes": trafficBytes,
		"logs": task.Logs + verifyMsg,
	})
	engine.SendWebhook(task, "镜像同步与校验完全成功。")
}
