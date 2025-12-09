#!/bin/bash
# 构建验证脚本 - 检查是否有外部CDN引用

set -e

DIST_DIR="dist"

echo "=== 检查构建产物中的外部引用 ==="

# 检查是否存在dist目录
if [ ! -d "$DIST_DIR" ]; then
    echo "错误: dist 目录不存在，请先运行 npm run build"
    exit 1
fi

# 查找所有HTML、JS、CSS文件中的外部URL
echo "检查 HTML 文件..."
if grep -r "https://" "$DIST_DIR"/*.html 2>/dev/null | grep -v "https://api" | grep -v "https://rgw" | grep -v "https://example"; then
    echo "⚠️  警告: 发现外部 HTTPS 引用"
else
    echo "✓ HTML 文件无外部引用"
fi

echo ""
echo "检查 JS 文件中的CDN引用..."
CDN_FILES=$(find "$DIST_DIR/assets" -name "*.js" -exec grep -l "cdn\.\|googleapis\|unpkg\|jsdelivr\|cdnjs" {} \; 2>/dev/null)
if [ -n "$CDN_FILES" ]; then
    echo "❌ 错误: 发现 CDN 引用"
    echo "$CDN_FILES"
    exit 1
else
    echo "✓ JS 文件无 CDN 引用"
fi

echo ""
echo "检查 CSS 文件中的外部字体..."
FONT_FILES=$(find "$DIST_DIR/assets" -name "*.css" -exec grep -l "fonts.googleapis\|fonts.gstatic" {} \; 2>/dev/null)
if [ -n "$FONT_FILES" ]; then
    echo "❌ 错误: 发现外部字体引用"
    echo "$FONT_FILES"
    exit 1
else
    echo "✓ CSS 文件无外部字体引用"
fi

echo ""
echo "=== 列出所有静态资源文件 ==="
find "$DIST_DIR" -type f | sort

echo ""
echo "=== 构建产物统计 ==="
echo "HTML 文件: $(find "$DIST_DIR" -name "*.html" | wc -l)"
echo "JS 文件: $(find "$DIST_DIR" -name "*.js" | wc -l)"
echo "CSS 文件: $(find "$DIST_DIR" -name "*.css" | wc -l)"
echo "总文件数: $(find "$DIST_DIR" -type f | wc -l)"
echo "总大小: $(du -sh "$DIST_DIR" | cut -f1)"

echo ""
echo "✅ 构建验证通过 - 所有资源均为本地打包"
