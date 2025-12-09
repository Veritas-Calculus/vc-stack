#!/bin/bash
# 初始化默认用户
# 用法: ./init-users.sh [host] [container_name]

HOST="${1:-10.31.0.3}"
CONTAINER="${2:-vc-stack-postgres}"

echo "初始化默认用户..."
echo "目标主机: ${HOST}"
echo "数据库容器: ${CONTAINER}"
echo ""

# admin 用户的密码: admin123 (bcrypt 哈希)
# shellcheck disable=SC2016
ADMIN_HASH='$2a$10$ELnoQYsFsfsvVwGxmzxtv.UPyXX3xkJxxuuHlsUoyv3sFi2kQX2Ui'

# testuser 用户的密码: test123 (bcrypt 哈希)
# shellcheck disable=SC2016
TEST_HASH='$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'

# shellcheck disable=SC2087,SC2029
cat << EOF | ssh user@"${HOST}" "docker exec -i \"${CONTAINER}\" psql -U vcstack -d vcstack"
-- 更新或插入 admin 用户
INSERT INTO users (username, email, password, first_name, last_name, is_active, is_admin, created_at, updated_at)
VALUES ('admin', 'admin@vcstack.com', '$ADMIN_HASH', 'Admin', 'User', true, true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (username) DO UPDATE SET
    password = EXCLUDED.password,
    is_active = EXCLUDED.is_active,
    is_admin = EXCLUDED.is_admin,
    updated_at = CURRENT_TIMESTAMP;

-- 更新或插入 testuser 用户
INSERT INTO users (username, email, password, first_name, last_name, is_active, is_admin, created_at, updated_at)
VALUES ('testuser', 'user@vcstack.com', '$TEST_HASH', 'Test', 'User', true, false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (username) DO UPDATE SET
    password = EXCLUDED.password,
    is_active = EXCLUDED.is_active,
    updated_at = CURRENT_TIMESTAMP;

-- 显示用户列表
SELECT id, username, email, is_active, is_admin FROM users ORDER BY id;
EOF

echo ""
echo "用户初始化完成！"
echo ""
echo "默认用户账号："
echo "  管理员："
echo "    用户名: admin"
echo "    密码: admin123"
echo ""
echo "  测试用户："
echo "    用户名: testuser"
echo "    密码: test123"
echo ""
echo "登录测试："
echo "  curl -X POST http://${HOST}:8080/api/v1/auth/login \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"username\":\"admin\",\"password\":\"admin123\"}'"
