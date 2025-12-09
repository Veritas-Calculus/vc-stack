#!/bin/bash
# VC Stack 回滚脚本
# 回滚到上一个备份版本

set -e

# 配置变量
REMOTE_HOST="10.31.0.3"
REMOTE_USER="user"
BACKUP_BASE_DIR="/opt/tiger/backup"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 列出可用的备份
list_backups() {
    log_info "可用的备份版本:"
    ssh "${REMOTE_USER}@${REMOTE_HOST}" "ls -lt ${BACKUP_BASE_DIR} 2>/dev/null || echo '无可用备份'"
}

# 回滚到指定备份
rollback() {
    local backup_dir=$1
    
    log_info "开始回滚到: ${backup_dir}"
    
    ssh "${REMOTE_USER}@${REMOTE_HOST}" << EOF
# 停止服务
sudo systemctl stop vc-controller
sudo systemctl stop vc-node
sleep 3

# 回滚二进制
if [ -d "${backup_dir}/bin" ]; then
    sudo rm -rf /opt/tiger/bin/*
    sudo cp -r ${backup_dir}/bin/* /opt/tiger/bin/
    sudo chmod +x /opt/tiger/bin/*
    echo "二进制文件已回滚"
fi

# 回滚配置
if [ -d "${backup_dir}/configs" ]; then
    sudo cp -r ${backup_dir}/configs/* /opt/tiger/configs/
    echo "配置文件已回滚"
fi

# 重启服务
sudo systemctl start vc-controller
sudo systemctl start vc-node

echo "回滚完成"
EOF
    
    log_info "回滚完成，请检查服务状态"
}

# 主函数
main() {
    echo ""
    echo "=========================================="
    echo "  VC Stack 回滚脚本"
    echo "=========================================="
    echo ""
    
    list_backups
    
    echo ""
    read -p "请输入要回滚的备份目录名称（或按 Ctrl+C 取消）: " backup_name
    
    if [ -z "$backup_name" ]; then
        log_error "未指定备份目录"
        exit 1
    fi
    
    rollback "${BACKUP_BASE_DIR}/${backup_name}"
}

main "$@"
