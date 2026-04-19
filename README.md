# Skopeo Sync Web

基于 `skopeo` 和 Go + Vue3 构建的 Docker 镜像同步管理 Web 系统，支持多架构镜像同步、图形化管理、失败重试、校验、流量统计以及多种 Webhook 通知（钉钉/企微/飞书）。

## 快速开始

本项目已提供完整的 `Dockerfile`，你可以通过 Docker 一键部署：

```bash
# 1. 克隆代码
git clone <repo_url> oci_sync
cd oci_sync

# 2. 构建镜像
docker build -t skopeo-sync-web:latest .

# 3. 运行容器 (请修改环境变量配置你的专属密码)
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -e ADMIN_USER=admin \
  -e ADMIN_PASSWORD=your_secure_password \
  -e JWT_SECRET=your_jwt_secret_key \
  --name skopeo-sync \
  skopeo-sync-web:latest
```

## 环境变量说明

| 变量名 | 默认值 | 描述 |
| ------ | ------ | ------ |
| `ADMIN_USER` | `admin` | 管理员后台登录账号 |
| `ADMIN_PASSWORD` | `admin123` | 管理员后台登录密码（**生产环境请务必修改**）|
| `JWT_SECRET` | `skopeo-sync-super-secret-key-123456` | JWT 加密密钥（**生产环境请务必修改**） |

**注意**: SQLite 数据库文件默认存放在工作目录中。建议通过 `-v` 挂载目录以持久化任务和配置数据（在宿主机映射到容器内的 `/app` 目录即可）。
