#!/bin/bash
# 基于 skopeo 的基础 Docker 镜像同步工具
# 用法: ./sync.sh <源镜像> <目标镜像> [最大重试次数]

SOURCE_IMAGE=${1}
DEST_IMAGE=${2}
MAX_RETRIES=${3:-3}

if [ -z "$SOURCE_IMAGE" ] || [ -z "$DEST_IMAGE" ]; then
    echo "用法: $0 <源镜像> <目标镜像> [最大重试次数]"
    echo "示例: $0 docker.io/library/nginx:latest myregistry.local/nginx:latest"
    exit 1
fi

echo "开始同步: ${SOURCE_IMAGE} -> ${DEST_IMAGE}"

# 执行同步的函数
sync_image() {
    local attempt=1
    while [ $attempt -le $MAX_RETRIES ]; do
        echo "正在进行第 $attempt 次同步尝试 (共 $MAX_RETRIES 次)..."
        # --all: 拉取多架构镜像
        skopeo copy --all "docker://${SOURCE_IMAGE}" "docker://${DEST_IMAGE}"
        
        if [ $? -eq 0 ]; then
            echo "同步成功完成。"
            return 0
        fi
        
        echo "同步失败。5秒后重试..."
        sleep 5
        attempt=$((attempt + 1))
    done
    
    echo "在 $MAX_RETRIES 次尝试后同步失败。"
    return 1
}

# 使用 digest 验证镜像一致性的函数
verify_image() {
    echo "正在验证镜像一致性..."
    # 获取源镜像的 digest
    SRC_DIGEST=$(skopeo inspect "docker://${SOURCE_IMAGE}" | grep -i '"Digest":' | head -n 1 | awk -F '"' '{print $4}')
    # 获取目标镜像的 digest
    DEST_DIGEST=$(skopeo inspect "docker://${DEST_IMAGE}" | grep -i '"Digest":' | head -n 1 | awk -F '"' '{print $4}')

    if [ -z "$SRC_DIGEST" ] || [ -z "$DEST_DIGEST" ]; then
        echo "验证跳过或失败：无法获取 digest。"
        return 1
    fi

    if [ "$SRC_DIGEST" == "$DEST_DIGEST" ]; then
        echo "验证成功：Digests 匹配 ($SRC_DIGEST)"
        return 0
    else
        echo "验证失败：源 ($SRC_DIGEST) != 目标 ($DEST_DIGEST)"
        return 1
    fi
}

sync_image
if [ $? -eq 0 ]; then
    verify_image
else
    exit 1
fi
