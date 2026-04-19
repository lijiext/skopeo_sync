FROM golang:alpine AS backend-builder
# 由于使用了 gorm sqlite，需要启用 CGO 并安装 gcc
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go env -w GOPROXY=https://goproxy.cn,direct && go mod download
COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -o skopeo-sync-api main.go

FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install --registry=https://registry.npmmirror.com
COPY frontend/ ./
RUN npm run build

FROM alpine:3.19
# 安装 skopeo 以及证书、时区数据和 sqlite 需要的 libc
RUN apk add --no-cache skopeo ca-certificates tzdata libc6-compat

WORKDIR /app
# 复制后端二进制
COPY --from=backend-builder /app/skopeo-sync-api .
# 复制前端静态文件 (Gin 后端需要提供静态文件服务)
COPY --from=frontend-builder /app/dist ./public

# 暴露端口
EXPOSE 8080

# 环境变量：管理员账号和密码（可以在 docker run 时覆盖）
ENV ADMIN_USER=admin
ENV ADMIN_PASSWORD=admin123
ENV JWT_SECRET=skopeo-sync-super-secret-key-123456
ENV GIN_MODE=release

# 添加启动脚本权限
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

# 运行服务
CMD ["./entrypoint.sh"]
