#!/bin/bash
set -e

REMOTE_HOST="10.31.0.3"
REMOTE_USER="user"

echo "Inspecting remote database on $REMOTE_USER@$REMOTE_HOST..."

ssh "${REMOTE_USER}@${REMOTE_HOST}" "bash -s" << 'EOF'
    set -e
    CONTAINER_NAME="vc-stack-postgres"
    DB_USER="vcstack"
    DB_NAME="vcstack"

    if docker ps | grep -q "vc-stack-postgres-simple"; then
        CONTAINER_NAME="vc-stack-postgres-simple"
    elif docker ps | grep -q "vc-stack-postgres"; then
        CONTAINER_NAME="vc-stack-postgres"
    fi

    echo "Found container: $CONTAINER_NAME"

    echo "--- Constraints on users table ---"
    docker exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT conname, contype FROM pg_constraint WHERE conrelid = 'users'::regclass;"

    echo "--- Indexes on users table ---"
    docker exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'users';"
EOF
