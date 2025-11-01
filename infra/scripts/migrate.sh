#!/bin/bash

# Echo Backend - Database Migration Script
set -e

echo "[INFO] Running database migrations..."

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '#' | awk '/=/ {print $1}')
fi

POSTGRES_HOST=${POSTGRES_HOST:-localhost}
POSTGRES_PORT=${POSTGRES_PORT:-5432}
POSTGRES_USER=${POSTGRES_USER:-echo}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-echo_password}
POSTGRES_DB=${POSTGRES_DB:-echo_db}

SERVICES=(
    "auth-service"
    "user-service"
    "message-service"
    "media-service"
    "notification-service"
    "analytics-service"
)

# Check if golang-migrate is installed
if ! command -v migrate &> /dev/null; then
    echo "[ERROR] golang-migrate is not installed"
    echo "   Install it: brew install golang-migrate"
    exit 1
fi

DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"

echo "[INFO] Migrating services..."

for service in "${SERVICES[@]}"; do
    MIGRATION_PATH="../../services/${service}/migrations"
    
    if [ -d "$MIGRATION_PATH" ]; then
        echo "   [INFO] Migrating $service..."
        
        # Check for postgres subdirectory
        if [ -d "${MIGRATION_PATH}/postgres" ]; then
            MIGRATION_PATH="${MIGRATION_PATH}/postgres"
        fi
        
        migrate -path "$MIGRATION_PATH" -database "$DATABASE_URL" up
        
        if [ $? -eq 0 ]; then
            echo "   [SUCCESS] $service migrated successfully"
        else
            echo "   [ERROR] Failed to migrate $service"
            exit 1
        fi
    else
        echo "   [WARN] Migration path not found for $service"
    fi
done

echo "[SUCCESS] All migrations completed successfully!"
