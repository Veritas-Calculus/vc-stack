#!/bin/bash
# 构建后处理脚本 - 更新novnc.html中的novnc bundle路径

set -e

DIST_DIR="dist"

# 查找novnc的实际文件名
NOVNC_FILE=$(find "$DIST_DIR/assets" -name "novnc-*.js" -type f | head -1)

if [ -z "$NOVNC_FILE" ]; then
    echo "警告: 未找到novnc bundle文件"
    exit 0
fi

# 提取文件名（不含路径）
NOVNC_FILENAME=$(basename "$NOVNC_FILE")

echo "找到 noVNC bundle: $NOVNC_FILENAME"

# 更新novnc.html中的引用
if [ -f "$DIST_DIR/novnc.html" ]; then
    # 使用sed替换占位符
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' "s|/assets/novnc.js|/assets/$NOVNC_FILENAME|g" "$DIST_DIR/novnc.html"
    else
        # Linux
        sed -i "s|/assets/novnc.js|/assets/$NOVNC_FILENAME|g" "$DIST_DIR/novnc.html"
    fi
    echo "✓ 已更新 novnc.html 中的引用"
else
    echo "警告: 未找到 $DIST_DIR/novnc.html"
fi
