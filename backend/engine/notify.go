package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/skopeo-sync/web/db"
	"github.com/skopeo-sync/web/models"
)

// SendWebhook 异步发送 Webhook 通知 (支持钉钉、企业微信、飞书、通用 Markdown)
func SendWebhook(task *models.SyncTask, msg string) {
	// 防止阻塞主调度流程，开个 Goroutine
	go func() {
		var urlConf, typeConf models.SysConfig

		// 查询数据库里是否配置了 webhook_url
		if err := db.DB.Where("key = ?", "webhook_url").First(&urlConf).Error; err != nil || urlConf.Value == "" {
			return // 未配置则不发送
		}

		// 获取 webhook 类型，默认为 general
		whType := "general"
		if err := db.DB.Where("key = ?", "webhook_type").First(&typeConf).Error; err == nil && typeConf.Value != "" {
			whType = typeConf.Value
		}

		var src, dest models.Registry
		db.DB.First(&src, task.SourceID)
		db.DB.First(&dest, task.DestID)

		srcUrl := src.URL + "/" + task.Image
		destUrl := dest.URL + "/" + task.DestImage
		if src.URL == "" {
			srcUrl = task.Image
		}
		if dest.URL == "" {
			destUrl = task.DestImage
		}

		// 针对不同平台构造适配的 Markdown
		var markdownContent string
		title := "📦 Skopeo 镜像同步通知"
		var payload interface{}

		switch whType {
		case "dingtalk":
			// 钉钉支持少量的 Markdown 标签。字体颜色通过 <font color=xxx> 实现
			statusColor := "<font color=\"#1890FF\">未知</font>"
			if task.Status == "success" {
				statusColor = "<font color=\"#52C41A\">✅ 成功</font>"
			} else if task.Status == "failed" {
				statusColor = "<font color=\"#FF4D4F\">❌ 失败</font>"
			}
			markdownContent = fmt.Sprintf("### %s\n- **任务 ID**: %d\n- **源镜像**: `%s`\n- **目标镜像**: `%s`\n- **状态**: %s\n- **详情**: %s\n- **时间**: %s",
				title, task.ID, srcUrl, destUrl, statusColor, msg, time.Now().Format("2006-01-02 15:04:05"))

			payload = map[string]interface{}{
				"msgtype": "markdown",
				"markdown": map[string]string{
					"title": title,
					"text":  markdownContent,
				},
			}

		case "wechat":
			// 企业微信同样支持 Markdown，颜色格式为 <font color="info"> 等内置的颜色变量 (info, comment, warning)
			statusColor := "<font color=\"comment\">未知</font>"
			if task.Status == "success" {
				statusColor = "<font color=\"info\">✅ 成功</font>"
			} else if task.Status == "failed" {
				statusColor = "<font color=\"warning\">❌ 失败</font>"
			}
			markdownContent = fmt.Sprintf("### %s\n> **任务 ID**: %d\n> **源镜像**: <font color=\"comment\">%s</font>\n> **目标镜像**: <font color=\"comment\">%s</font>\n> **状态**: %s\n> **详情**: %s\n> **时间**: %s",
				title, task.ID, srcUrl, destUrl, statusColor, msg, time.Now().Format("2006-01-02 15:04:05"))

			payload = map[string]interface{}{
				"msgtype": "markdown",
				"markdown": map[string]string{
					"content": markdownContent,
				},
			}

		case "feishu":
			// 飞书的互动卡片中 markdown 标签支持有限。不支持原生的 <font color="xxx"> 这种 HTML 语法。
			// 因此在飞书的卡片中，我们需要使用飞书自带的纯 markdown（无 HTML 颜色标签）或者直接不带颜色，只带 emoji。
			statusEmoji := "❓ 未知"
			if task.Status == "success" {
				statusEmoji = "✅ 成功"
			} else if task.Status == "failed" {
				statusEmoji = "❌ 失败"
			}
			
			// 飞书卡片里的 markdown 换行直接使用 \n 即可
			markdownContent = fmt.Sprintf("**任务 ID**: %d\n**源镜像**: `%s`\n**目标镜像**: `%s`\n**状态**: %s\n**详情**: %s\n**时间**: %s",
				task.ID, srcUrl, destUrl, statusEmoji, msg, time.Now().Format("2006-01-02 15:04:05"))

			// 发送飞书互动卡片 V2 格式 (Interactive Message)
			payload = map[string]interface{}{
				"msg_type": "interactive",
				"card": map[string]interface{}{
					"config": map[string]interface{}{
						"wide_screen_mode": true,
					},
					"header": map[string]interface{}{
						"template": func() string {
							if task.Status == "success" {
								return "green"
							} else if task.Status == "failed" {
								return "red"
							}
							return "blue"
						}(),
						"title": map[string]interface{}{
							"content": title,
							"tag":     "plain_text",
						},
					},
					"elements": []map[string]interface{}{
						{
							"tag": "markdown",
							"content": markdownContent,
						},
					},
				},
			}

		default:
			// 通用 JSON 格式 (如自建服务)
			payload = map[string]interface{}{
				"task_id":    task.ID,
				"image":      srcUrl,
				"dest_image": destUrl,
				"status":     task.Status,
				"message":    msg,
				"time":       time.Now().Format(time.RFC3339),
			}
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			log.Printf("[Webhook] JSON 序列化失败: %v", err)
			return
		}

		resp, err := http.Post(urlConf.Value, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("[Webhook] 发送失败: %v", err)
			return
		}
		defer resp.Body.Close()

		log.Printf("[Webhook] 通知已发送至 %s (%s)，响应状态码: %d", whType, urlConf.Value, resp.StatusCode)
	}()
}
