#!/bin/bash

# VC Stack 数据库管理脚本
# 用于启动、停止和管理开发数据库

DB_COMPOSE_FILE="docker-compose.dev.yml"

case ${1:-"help"} in
    "start")
        echo "🚀 启动开发数据库..."
        docker-compose -f $DB_COMPOSE_FILE up -d
        echo "✅ 数据库启动完成!"
        echo "📊 PostgreSQL: localhost:5432"
        echo "🔴 Redis: localhost:6379"
        echo "📋 数据库信息:"
        echo "   - 数据库名: vcstack"
        echo "   - 用户名: vcstack"
        echo "   - 密码: vcstack123"
        ;;
    "stop")
        echo "🛑 停止开发数据库..."
        docker-compose -f $DB_COMPOSE_FILE down
        echo "✅ 数据库已停止"
        ;;
    "restart")
        echo "🔄 重启开发数据库..."
        docker-compose -f $DB_COMPOSE_FILE restart
        echo "✅ 数据库重启完成"
        ;;
    "logs")
        echo "📋 查看数据库日志..."
        docker-compose -f $DB_COMPOSE_FILE logs -f "${2:-""}"
        ;;
    "status")
        echo "📊 数据库状态:"
        docker-compose -f $DB_COMPOSE_FILE ps
        ;;
    "clean")
        echo "🧹 清理数据库数据 (危险操作)..."
        read -p "确定要删除所有数据吗? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            docker-compose -f $DB_COMPOSE_FILE down -v
            echo "✅ 数据库数据已清理"
        else
            echo "❌ 操作已取消"
        fi
        ;;
    "psql")
        echo "🐘 连接到PostgreSQL..."
        docker exec -it vc-stack-postgres psql -U vcstack -d vcstack
        ;;
    "redis")
        echo "🔴 连接到Redis..."
        docker exec -it vc-stack-redis redis-cli
        ;;
    "backup")
        BACKUP_FILE="backup_$(date +%Y%m%d_%H%M%S).sql"
        echo "💾 备份数据库到 $BACKUP_FILE..."
        docker exec vc-stack-postgres pg_dump -U vcstack vcstack > "$BACKUP_FILE"
        echo "✅ 备份完成: $BACKUP_FILE"
        ;;
    "restore")
        if [ -z "$2" ]; then
            echo "❌ 请指定备份文件"
            echo "用法: $0 restore backup_file.sql"
            exit 1
        fi
        echo "📥 从 $2 恢复数据库..."
        docker exec -i vc-stack-postgres psql -U vcstack vcstack < "$2"
        echo "✅ 恢复完成"
        ;;
    "help"|*)
        echo "🎯 VC Stack 数据库管理工具"
        echo ""
        echo "用法: $0 <command> [options]"
        echo ""
        echo "命令:"
        echo "  start     启动数据库容器"
        echo "  stop      停止数据库容器"
        echo "  restart   重启数据库容器"
        echo "  status    查看容器状态"
        echo "  logs      查看日志 (可选: logs postgres|redis)"
        echo "  clean     清理所有数据 (危险!)"
        echo "  psql      连接PostgreSQL命令行"
        echo "  redis     连接Redis命令行"
        echo "  backup    备份数据库"
        echo "  restore   恢复数据库 (需要指定备份文件)"
        echo "  help      显示此帮助信息"
        echo ""
        echo "示例:"
        echo "  $0 start          # 启动数据库"
        echo "  $0 psql           # 连接数据库"
        echo "  $0 logs postgres  # 查看PostgreSQL日志"
        echo "  $0 backup         # 备份数据库"
        ;;
esac
