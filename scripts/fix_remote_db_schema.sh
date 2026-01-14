#!/bin/bash
set -e

REMOTE_HOST="10.31.0.3"
REMOTE_USER="user"
MIGRATION_FILE="migrations/004_add_iam_policies.sql"
REMOTE_TMP_DIR="/tmp"

echo "Deploying migration to $REMOTE_USER@$REMOTE_HOST..."

# Check if migration file exists locally
if [ ! -f "$MIGRATION_FILE" ]; then
    echo "Error: Migration file $MIGRATION_FILE not found locally."
    exit 1
fi

# Copy migration file to remote
echo "Copying $MIGRATION_FILE to remote..."
scp "$MIGRATION_FILE" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_TMP_DIR}/iam_migration.sql"

# Execute migration on remote
echo "Executing migration on remote..."
ssh "${REMOTE_USER}@${REMOTE_HOST}" "bash -s" << 'EOF'
    set -e
    CONTAINER_NAME="vc-stack-postgres"
    DB_USER="vcstack"
    DB_NAME="vcstack"
    MIGRATION_SQL="/tmp/iam_migration.sql"

    echo "Checking for database container..."
    if docker ps | grep -q "vc-stack-postgres-simple"; then
        CONTAINER_NAME="vc-stack-postgres-simple"
    elif docker ps | grep -q "vc-stack-postgres"; then
        CONTAINER_NAME="vc-stack-postgres"
    else
        echo "Error: Could not find running postgres container"
        exit 1
    fi

    echo "Found container: $CONTAINER_NAME"
    echo "Applying migration..."

    cat "$MIGRATION_SQL" | docker exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME"

    echo "Migration applied successfully."
    rm "$MIGRATION_SQL"
EOF

echo "Done."
