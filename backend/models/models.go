package models

import "gorm.io/gorm"

// Registry 镜像仓库配置模型
type Registry struct {
	gorm.Model
	Name     string `json:"name"`     // 名称，例如: Docker Hub
	URL      string `json:"url"`      // 地址，例如: docker.io
	Username string `json:"username"` // 用户名
	Password string `json:"password"` // 密码 (生产环境应加密存储)
}

// SyncTask 同步任务模型
type SyncTask struct {
	gorm.Model
	SourceID     uint   `json:"source_id"`     // 源仓库 ID
	DestID       uint   `json:"dest_id"`       // 目标仓库 ID
	Image        string `json:"image"`         // 源镜像名称及标签
	DestImage    string `json:"dest_image"`    // 目标镜像名称及标签 (新增)
	Status       string `json:"status"`        // 状态 (pending, running, success, failed)
	TrafficBytes int64  `json:"traffic_bytes"` // 消耗流量 (字节)
	Retries      int    `json:"retries"`       // 重试次数设定
	CurrentRetry int    `json:"current_retry"` // 当前已重试次数 (新增)
	Logs         string `json:"logs"`          // 详细执行日志 (新增)
}

// SysConfig 系统全局配置 (如 Webhook)
type SysConfig struct {
	gorm.Model
	Key   string `json:"key" gorm:"uniqueIndex"`
	Value string `json:"value"`
}
