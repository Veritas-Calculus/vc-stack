#!/bin/bash
# VC Stack 配置验证脚本
# 用于检查配置文件的完整性和正确性

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "======================================"
echo "VC Stack 配置验证工具"
echo "======================================"
echo ""

# 检查配置文件是否存在
check_file_exists() {
    local file=$1
    local required=$2
    
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC} 找到: $file"
        return 0
    else
        if [ "$required" = "true" ]; then
            echo -e "${RED}✗${NC} 缺失 (必需): $file"
            return 1
        else
            echo -e "${YELLOW}⚠${NC} 缺失 (可选): $file"
            return 0
        fi
    fi
}

# 检查 YAML 语法
check_yaml_syntax() {
    local file=$1
    
    if ! command -v python3 &> /dev/null; then
        echo -e "${YELLOW}⚠${NC} Python3 未安装，跳过 YAML 语法检查"
        return 0
    fi
    
    if python3 -c "import yaml; yaml.safe_load(open('$file'))" 2>/dev/null; then
        echo -e "${GREEN}✓${NC} YAML 语法正确: $file"
        return 0
    else
        echo -e "${RED}✗${NC} YAML 语法错误: $file"
        return 1
    fi
}

# 检查配置文件权限
check_file_permissions() {
    local file=$1
    local perms=$(stat -f "%OLp" "$file" 2>/dev/null || stat -c "%a" "$file" 2>/dev/null)
    
    if [ "$perms" = "600" ] || [ "$perms" = "400" ]; then
        echo -e "${GREEN}✓${NC} 权限正确 ($perms): $file"
        return 0
    else
        echo -e "${YELLOW}⚠${NC} 建议权限 600: $file (当前: $perms)"
        return 0
    fi
}

# 检查数据库连接
check_database() {
    local host=$1
    local port=$2
    local db=$3
    local user=$4
    
    echo "检查数据库连接: $user@$host:$port/$db"
    
    if command -v psql &> /dev/null; then
        if PGPASSWORD=$DATABASE_PASSWORD psql -h "$host" -p "$port" -U "$user" -d "$db" -c "SELECT 1;" &>/dev/null; then
            echo -e "${GREEN}✓${NC} 数据库连接成功"
            return 0
        else
            echo -e "${RED}✗${NC} 数据库连接失败"
            return 1
        fi
    else
        echo -e "${YELLOW}⚠${NC} psql 未安装，跳过数据库连接检查"
        return 0
    fi
}

# 检查端口占用
check_port() {
    local port=$1
    local name=$2
    
    if lsof -i :"$port" &>/dev/null || netstat -an | grep ":$port " &>/dev/null; then
        echo -e "${YELLOW}⚠${NC} 端口 $port ($name) 已被占用"
        return 1
    else
        echo -e "${GREEN}✓${NC} 端口 $port ($name) 可用"
        return 0
    fi
}

# 检查服务依赖
check_service() {
    local service=$1
    
    if systemctl is-active --quiet "$service" 2>/dev/null; then
        echo -e "${GREEN}✓${NC} 服务运行中: $service"
        return 0
    else
        echo -e "${YELLOW}⚠${NC} 服务未运行: $service"
        return 0
    fi
}

echo "=== 1. 检查配置文件存在性 ==="
echo ""

ERRORS=0

# Controller 配置
if check_file_exists "/etc/vc-stack/controller.yaml" "false"; then
    check_yaml_syntax "/etc/vc-stack/controller.yaml"
    check_file_permissions "/etc/vc-stack/controller.yaml"
fi

if check_file_exists "/etc/vc-stack/controller.env" "false"; then
    check_file_permissions "/etc/vc-stack/controller.env"
fi

# Node 配置
if check_file_exists "/etc/vc-stack/node.yaml" "false"; then
    check_yaml_syntax "/etc/vc-stack/node.yaml"
    check_file_permissions "/etc/vc-stack/node.yaml"
fi

if check_file_exists "/etc/vc-stack/node.env" "false"; then
    check_file_permissions "/etc/vc-stack/node.env"
fi

# Systemd 服务文件
check_file_exists "/etc/systemd/system/vc-controller.service" "false"
check_file_exists "/etc/systemd/system/vc-node.service" "false"

echo ""
echo "=== 2. 检查二进制文件 ==="
echo ""

check_file_exists "/opt/vc-stack/bin/vc-controller" "false"
check_file_exists "/opt/vc-stack/bin/vc-node" "false"
check_file_exists "/opt/vc-stack/bin/vcctl" "false"

echo ""
echo "=== 3. 检查端口可用性 ==="
echo ""

check_port 8080 "Controller API"
check_port 9090 "Controller Metrics"
check_port 9091 "Node Metrics"
check_port 5432 "PostgreSQL"

echo ""
echo "=== 4. 检查依赖服务 ==="
echo ""

check_service "postgresql"
check_service "libvirtd"
check_service "ovs-vswitchd"

echo ""
echo "=== 5. 检查 VC Stack 服务 ==="
echo ""

check_service "vc-controller"
check_service "vc-node"

echo ""
echo "======================================"
if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}验证完成！未发现严重问题。${NC}"
else
    echo -e "${YELLOW}验证完成，发现 $ERRORS 个问题需要处理。${NC}"
fi
echo "======================================"
