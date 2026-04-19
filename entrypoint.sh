#!/bin/sh
set -e

# 在 docker 容器启动前，做一些环境准备工作，如果需要的话
echo "Starting Skopeo Sync System..."
echo "Admin User: ${ADMIN_USER:-admin}"

exec ./skopeo-sync-api
