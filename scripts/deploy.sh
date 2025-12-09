#!/bin/bash
# VC Stack 部署脚本
# 将构建好的二进制和配置文件部署到远程服务器

set -e

# 配置变量
REMOTE_HOST="10.31.0.3"
REMOTE_USER="user"
REMOTE_BIN_DIR="/opt/tiger/bin"
REMOTE_CONFIG_DIR="/opt/tiger/configs"
REMOTE_WEB_DIR="/opt/tiger/web"
REMOTE_LOG_DIR="/var/log/vc-stack"
REMOTE_DATA_DIR="/var/lib/vc-stack"

# 本地目录
LOCAL_BIN_DIR="./bin"
LOCAL_WEB_DIR="./web/console/dist"
LOCAL_CONFIG_DIR="./configs"

# 构建信息
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 SSH 连接
check_ssh() {
    log_info "检查 SSH 连接到 ${REMOTE_USER}@${REMOTE_HOST}..."
    if ssh -o ConnectTimeout=5 "${REMOTE_USER}@${REMOTE_HOST}" "echo 'SSH 连接成功'" &>/dev/null; then
        log_info "SSH 连接正常"
        return 0
    else
        log_error "无法连接到远程主机"
        return 1
    fi
}

# 构建二进制文件（linux/amd64）
build_binaries() {
    log_info "开始构建 linux/amd64 二进制文件..."
    
    # 清理旧的构建
    rm -rf "${LOCAL_BIN_DIR}"
    mkdir -p "${LOCAL_BIN_DIR}"
    
    # 构建所有服务
    GOOS=linux GOARCH=amd64 make build
    
    if [ $? -eq 0 ]; then
        log_info "二进制构建完成"
        ls -lh "${LOCAL_BIN_DIR}"
    else
        log_error "构建失败"
        exit 1
    fi
}

# 构建前端
build_frontend() {
    log_info "开始构建前端..."
    
    cd web/console
    
    # 安装依赖（如果需要）
    if [ ! -d "node_modules" ]; then
        log_info "安装前端依赖..."
        npm install
    fi
    
    # 构建前端
    npm run build:verify
    
    if [ $? -eq 0 ]; then
        log_info "前端构建完成"
        cd ../..
    else
        log_error "前端构建失败"
        cd ../..
        exit 1
    fi
}

# 停止远程服务
stop_remote_services() {
    log_info "停止远程服务..."
    
    ssh "${REMOTE_USER}@${REMOTE_HOST}" << 'EOF'
# 停止 systemd 服务
sudo systemctl stop vc-controller 2>/dev/null || true
sudo systemctl stop vc-node 2>/dev/null || true

# 查找并停止所有相关进程
sudo pkill -f vc-controller || true
sudo pkill -f vc-node || true
sudo pkill -f vcctl || true

# 等待进程完全停止
sleep 3

# 确认进程已停止
if pgrep -f "vc-controller|vc-node" > /dev/null; then
    echo "警告: 仍有进程在运行"
    ps aux | grep -E "vc-controller|vc-node" | grep -v grep
    echo "强制终止..."
    sudo pkill -9 -f vc-controller || true
    sudo pkill -9 -f vc-node || true
    sleep 2
fi

echo "所有服务已停止"
EOF
    
    log_info "远程服务已停止"
}

# 创建远程目录
create_remote_dirs() {
    log_info "创建远程目录..."
    
    ssh "${REMOTE_USER}@${REMOTE_HOST}" << EOF
sudo mkdir -p ${REMOTE_BIN_DIR}
sudo mkdir -p ${REMOTE_CONFIG_DIR}
sudo mkdir -p ${REMOTE_WEB_DIR}
sudo mkdir -p ${REMOTE_LOG_DIR}
sudo mkdir -p ${REMOTE_DATA_DIR}
sudo mkdir -p /etc/systemd/system

# 设置权限
sudo chown -R ${REMOTE_USER}:${REMOTE_USER} ${REMOTE_LOG_DIR}
sudo chown -R ${REMOTE_USER}:${REMOTE_USER} ${REMOTE_DATA_DIR}

echo "远程目录创建完成"
EOF
}

# 备份旧版本
backup_old_version() {
    log_info "备份旧版本..."
    
    ssh "${REMOTE_USER}@${REMOTE_HOST}" << EOF
BACKUP_DIR="/opt/tiger/backup/\$(date +%Y%m%d_%H%M%S)"
sudo mkdir -p "\${BACKUP_DIR}"

# 备份二进制
if [ -d "${REMOTE_BIN_DIR}" ]; then
    sudo cp -r ${REMOTE_BIN_DIR} "\${BACKUP_DIR}/" 2>/dev/null || true
fi

# 备份配置
if [ -d "${REMOTE_CONFIG_DIR}" ]; then
    sudo cp -r ${REMOTE_CONFIG_DIR} "\${BACKUP_DIR}/" 2>/dev/null || true
fi

echo "备份完成: \${BACKUP_DIR}"
EOF
}

# 部署二进制文件
deploy_binaries() {
    log_info "部署二进制文件..."
    
    # 先复制到临时目录
    scp -r "${LOCAL_BIN_DIR}"/* "${REMOTE_USER}@${REMOTE_HOST}:/tmp/"
    
    # 然后使用 sudo 移动到目标目录并设置权限
    ssh "${REMOTE_USER}@${REMOTE_HOST}" << EOF
sudo mv /tmp/vc-controller ${REMOTE_BIN_DIR}/ 2>/dev/null || true
sudo mv /tmp/vc-node ${REMOTE_BIN_DIR}/ 2>/dev/null || true
sudo mv /tmp/vcctl ${REMOTE_BIN_DIR}/ 2>/dev/null || true

sudo chmod +x ${REMOTE_BIN_DIR}/vc-controller
sudo chmod +x ${REMOTE_BIN_DIR}/vc-node
sudo chmod +x ${REMOTE_BIN_DIR}/vcctl

echo "二进制文件部署完成"
ls -lh ${REMOTE_BIN_DIR}
EOF
    
    log_info "二进制文件部署完成"
}

# 部署配置文件
deploy_configs() {
    log_info "配置通过环境变量管理，跳过配置文件部署..."
    
    # 创建配置目录（如果需要）
    ssh "${REMOTE_USER}@${REMOTE_HOST}" << 'EOF'
sudo mkdir -p /opt/tiger/configs
echo "配置目录已创建"
EOF
    
    log_info "配置文件部署完成"
}

# 部署前端
deploy_frontend() {
    log_info "部署前端..."
    
    # 先复制到临时目录
    ssh "${REMOTE_USER}@${REMOTE_HOST}" "mkdir -p /tmp/vc-web-dist"
    scp -r "${LOCAL_WEB_DIR}"/* "${REMOTE_USER}@${REMOTE_HOST}:/tmp/vc-web-dist/"
    
    # 清空远程 web 目录并复制
    ssh "${REMOTE_USER}@${REMOTE_HOST}" << EOF
sudo rm -rf ${REMOTE_WEB_DIR}/*
sudo cp -r /tmp/vc-web-dist/* ${REMOTE_WEB_DIR}/
sudo rm -rf /tmp/vc-web-dist
echo "前端部署完成"
EOF
    
    log_info "前端部署完成"
}

# 部署 systemd 服务文件
deploy_systemd_services() {
    log_info "部署 systemd 服务文件..."
    
    # 创建临时服务文件
    cat > /tmp/vc-controller.service << 'EOF'
[Unit]
Description=VC Stack Controller - Control Plane Services
Documentation=https://github.com/Veritas-Calculus/vc-stack
After=network-online.target docker.service
Wants=network-online.target
Requires=docker.service

[Service]
Type=simple
User=user
Group=user

WorkingDirectory=/opt/tiger
ExecStart=/opt/tiger/bin/vc-controller

Restart=on-failure
RestartSec=10s

LimitNOFILE=65535
LimitNPROC=4096

Environment="DB_HOST=localhost"
Environment="DB_PORT=5432"
Environment="DB_NAME=vcstack"
Environment="DB_USER=vcstack"
Environment="DB_PASS=vcstack123"
Environment="VC_CONTROLLER_PORT=8080"

StandardOutput=journal
StandardError=journal
SyslogIdentifier=vc-controller

[Install]
WantedBy=multi-user.target
EOF

    cat > /tmp/vc-node.service << 'EOF'
[Unit]
Description=VC Stack Node - Compute and Network Agent
Documentation=https://github.com/Veritas-Calculus/vc-stack
After=network-online.target libvirtd.service
Wants=network-online.target
Requires=libvirtd.service

[Service]
Type=simple
User=root
Group=root

WorkingDirectory=/opt/tiger
ExecStart=/opt/tiger/bin/vc-node

Restart=on-failure
RestartSec=10s

LimitNOFILE=65535
LimitNPROC=8192

Environment="DB_HOST=localhost"
Environment="DB_PORT=5432"
Environment="DB_NAME=vcstack"
Environment="DB_USER=vcstack"
Environment="DB_PASS=vcstack123"
Environment="VC_CONTROLLER_PORT=8080"

Environment="CONTROLLER_URL=http://localhost:8080"
Environment="NODE_NAME=node-1"
Environment="NODE_PORT=8091"

StandardOutput=journal
StandardError=journal
SyslogIdentifier=vc-node

[Install]
WantedBy=multi-user.target
EOF

    # 复制到远程
    scp /tmp/vc-controller.service "${REMOTE_USER}@${REMOTE_HOST}:/tmp/"
    scp /tmp/vc-node.service "${REMOTE_USER}@${REMOTE_HOST}:/tmp/"
    
    # 安装服务文件
    ssh "${REMOTE_USER}@${REMOTE_HOST}" << 'EOFREMOTE'
sudo mv /tmp/vc-controller.service /etc/systemd/system/
sudo mv /tmp/vc-node.service /etc/systemd/system/

sudo systemctl daemon-reload
echo "systemd 服务文件已更新"
EOFREMOTE
    
    # 清理本地临时文件
    rm -f /tmp/vc-controller.service /tmp/vc-node.service
    
    log_info "systemd 服务文件部署完成"
}

# 部署 nginx 配置
deploy_nginx_config() {
    log_info "部署 nginx 配置文件..."
    
    if [ ! -f "configs/nginx/vc-stack.conf" ]; then
        log_warn "nginx 配置文件不存在，跳过部署"
        return 0
    fi
    
    scp configs/nginx/vc-stack.conf "${REMOTE_USER}@${REMOTE_HOST}:/tmp/vc-stack.conf"
    
    ssh "${REMOTE_USER}@${REMOTE_HOST}" << 'EOF'
# 部署 nginx 配置
sudo mv /tmp/vc-stack.conf /etc/nginx/sites-enabled/vc-stack.conf

# 测试 nginx 配置
if ! sudo nginx -t &>/dev/null; then
    echo "错误: nginx 配置文件有问题"
    exit 1
fi

# 重新加载 nginx
sudo systemctl reload nginx

echo "nginx 配置已更新"
EOF
    
    log_info "nginx 配置文件部署完成"
}

# 检查数据库容器
check_database() {
    log_info "检查数据库容器..."
    
    ssh "${REMOTE_USER}@${REMOTE_HOST}" << 'EOF'
if docker ps | grep -q vc-stack-postgres; then
    echo "数据库容器正在运行"
else
    echo "警告: 数据库容器未运行"
    echo "请确保 vc-stack-postgres 容器正常运行"
fi
EOF
}

# 启动服务
start_services() {
    log_info "启动服务..."
    
    ssh "${REMOTE_USER}@${REMOTE_HOST}" << 'EOF'
# 启动 controller
sudo systemctl enable vc-controller
sudo systemctl start vc-controller

# 等待 controller 启动
sleep 5

# 检查 controller 状态
if sudo systemctl is-active --quiet vc-controller; then
    echo "vc-controller 启动成功"
else
    echo "vc-controller 启动失败，查看日志:"
    sudo journalctl -u vc-controller -n 20 --no-pager
    exit 1
fi

# 启动 node
sudo systemctl enable vc-node
sudo systemctl start vc-node

# 等待 node 启动
sleep 5

# 检查 node 状态
if sudo systemctl is-active --quiet vc-node; then
    echo "vc-node 启动成功"
else
    echo "vc-node 启动失败，查看日志:"
    sudo journalctl -u vc-node -n 20 --no-pager
    exit 1
fi

echo ""
echo "所有服务启动完成"
echo ""
echo "服务状态:"
sudo systemctl status vc-controller --no-pager -l
echo ""
sudo systemctl status vc-node --no-pager -l
EOF
    
    log_info "服务启动完成"
}

# 验证部署
verify_deployment() {
    log_info "验证部署..."
    
    ssh "${REMOTE_USER}@${REMOTE_HOST}" << 'EOF'
echo ""
echo "=== 部署验证 ==="
echo ""

# 检查二进制版本
echo "二进制文件:"
ls -lh /opt/tiger/bin/

echo ""
echo "服务状态:"
sudo systemctl is-active vc-controller && echo "✓ vc-controller: 运行中" || echo "✗ vc-controller: 未运行"
sudo systemctl is-active vc-node && echo "✓ vc-node: 运行中" || echo "✗ vc-node: 未运行"

echo ""
echo "监听端口:"
sudo netstat -tlnp | grep -E "8080|8091" || true

echo ""
echo "前端文件:"
ls -lh /opt/tiger/web/ | head -5

echo ""
echo "日志文件:"
ls -lh /var/log/vc-stack/ 2>/dev/null || echo "暂无日志文件"

echo ""
echo "=== 部署验证完成 ==="
EOF
}

# 显示部署信息
show_deployment_info() {
    log_info "部署信息:"
    echo ""
    echo "  版本: ${VERSION}"
    echo "  提交: ${COMMIT}"
    echo "  构建时间: ${BUILD_TIME}"
    echo "  远程主机: ${REMOTE_USER}@${REMOTE_HOST}"
    echo ""
    echo "  访问地址:"
    echo "    - Controller API: http://${REMOTE_HOST}:8080"
    echo "    - Node API: http://${REMOTE_HOST}:8091"
    echo "    - Web Console: http://${REMOTE_HOST}:8080"
    echo "    - Monitoring: http://${REMOTE_HOST}:9090"
    echo ""
    echo "  配置文件:"
    echo "    - Controller: ${REMOTE_CONFIG_DIR}/vc-controller.yaml"
    echo "    - Node: ${REMOTE_CONFIG_DIR}/vc-node.yaml"
    echo ""
    echo "  日志查看:"
    echo "    - Controller: sudo journalctl -u vc-controller -f"
    echo "    - Node: sudo journalctl -u vc-node -f"
    echo ""
}

# 主函数
main() {
    echo ""
    echo "=========================================="
    echo "  VC Stack 部署脚本"
    echo "=========================================="
    echo ""
    
    # 检查 SSH 连接
    check_ssh || exit 1
    
    # 构建
    log_info "步骤 1/10: 构建二进制文件"
    build_binaries
    
    log_info "步骤 2/10: 构建前端"
    build_frontend
    
    # 部署
    log_info "步骤 3/10: 停止远程服务"
    stop_remote_services
    
    log_info "步骤 4/10: 创建远程目录"
    create_remote_dirs
    
    log_info "步骤 5/10: 备份旧版本"
    backup_old_version
    
    log_info "步骤 6/10: 部署二进制文件"
    deploy_binaries
    
    log_info "步骤 7/10: 部署配置文件"
    deploy_configs
    
    log_info "步骤 8/10: 部署前端"
    deploy_frontend
    
    log_info "步骤 9/11: 部署 systemd 服务"
    deploy_systemd_services
    
    log_info "步骤 10/11: 部署 nginx 配置"
    deploy_nginx_config
    
    log_info "步骤 11/11: 启动服务"
    check_database
    start_services
    
    # 验证
    verify_deployment
    
    # 显示部署信息
    echo ""
    show_deployment_info
    
    log_info "部署完成！"
}

# 运行主函数
main "$@"
