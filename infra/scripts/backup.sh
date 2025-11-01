#!/bin/bash

# Echo Backend - Database Backup Script
set -e

echo "[INFO] Starting database backup..."

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '#' | awk '/=/ {print $1}')
fi

POSTGRES_HOST=${POSTGRES_HOST:-localhost}
POSTGRES_PORT=${POSTGRES_PORT:-5432}
POSTGRES_USER=${POSTGRES_USER:-echo}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-echo_password}
POSTGRES_DB=${POSTGRES_DB:-echo_db}

# Backup directory
BACKUP_DIR="../../backups"
mkdir -p $BACKUP_DIR

# Timestamp for backup file
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Define schemas to backup
SCHEMAS=("auth" "users" "messages" "media" "notifications" "analytics" "location")

# Backup type: full or schema
BACKUP_TYPE=${1:-full}

if [ "$BACKUP_TYPE" = "full" ]; then
    BACKUP_FILE="${BACKUP_DIR}/echo_backup_full_${TIMESTAMP}.sql"
    echo "[INFO] Creating full database backup: $(basename $BACKUP_FILE).gz"
    
    # Create full backup
    PGPASSWORD=$POSTGRES_PASSWORD pg_dump \
        -h $POSTGRES_HOST \
        -p $POSTGRES_PORT \
        -U $POSTGRES_USER \
        -d $POSTGRES_DB \
        -F p \
        --no-owner \
        --no-acl \
        --verbose \
        -f $BACKUP_FILE
    
    # Compress backup
    gzip $BACKUP_FILE
    BACKUP_FILE_COMPRESSED="${BACKUP_FILE}.gz"
    
    # Get file size
    FILESIZE=$(ls -lh $BACKUP_FILE_COMPRESSED | awk '{print $5}')
    
    echo "[SUCCESS] Full backup created successfully!"
    echo "   File: $(basename $BACKUP_FILE_COMPRESSED)"
    echo "   Size: $FILESIZE"
    
elif [ "$BACKUP_TYPE" = "schema" ]; then
    echo "[INFO] Creating schema-specific backups..."
    
    # Create schema directory
    SCHEMA_BACKUP_DIR="${BACKUP_DIR}/schemas_${TIMESTAMP}"
    mkdir -p $SCHEMA_BACKUP_DIR
    
    # Backup each schema separately
    for SCHEMA in "${SCHEMAS[@]}"; do
        echo "[INFO] Backing up schema: $SCHEMA"
        SCHEMA_FILE="${SCHEMA_BACKUP_DIR}/${SCHEMA}_${TIMESTAMP}.sql"
        
        PGPASSWORD=$POSTGRES_PASSWORD pg_dump \
            -h $POSTGRES_HOST \
            -p $POSTGRES_PORT \
            -U $POSTGRES_USER \
            -d $POSTGRES_DB \
            -n $SCHEMA \
            -F p \
            --no-owner \
            --no-acl \
            -f $SCHEMA_FILE
        
        # Compress schema backup
        gzip $SCHEMA_FILE
        echo "   ✓ ${SCHEMA} schema backed up"
    done
    
    # Create archive of all schemas
    cd $BACKUP_DIR
    tar -czf "schemas_${TIMESTAMP}.tar.gz" "schemas_${TIMESTAMP}"
    rm -rf "schemas_${TIMESTAMP}"
    
    FILESIZE=$(ls -lh "schemas_${TIMESTAMP}.tar.gz" | awk '{print $5}')
    echo "[SUCCESS] Schema backups created successfully!"
    echo "   File: schemas_${TIMESTAMP}.tar.gz"
    echo "   Size: $FILESIZE"
    
else
    echo "[ERROR] Invalid backup type. Use 'full' or 'schema'"
    exit 1
fi

# Clean up old backups (keep last 7 days)
echo "[INFO] Cleaning up old backups..."
find $BACKUP_DIR -name "echo_backup_full_*.sql.gz" -mtime +7 -delete
find $BACKUP_DIR -name "schemas_*.tar.gz" -mtime +7 -delete

# Count remaining backups
FULL_BACKUP_COUNT=$(ls -1 $BACKUP_DIR/echo_backup_full_*.sql.gz 2>/dev/null | wc -l | tr -d ' ')
SCHEMA_BACKUP_COUNT=$(ls -1 $BACKUP_DIR/schemas_*.tar.gz 2>/dev/null | wc -l | tr -d ' ')
echo "[INFO] Total full backups: $FULL_BACKUP_COUNT"
echo "[INFO] Total schema backups: $SCHEMA_BACKUP_COUNT"

# Optional: Upload to cloud storage (uncomment and configure)
# echo "[INFO] Uploading to cloud storage..."
# if [ "$BACKUP_TYPE" = "full" ]; then
#     gsutil cp $BACKUP_FILE_COMPRESSED gs://echo-backups/
#     # aws s3 cp $BACKUP_FILE_COMPRESSED s3://echo-backups/
# else
#     gsutil cp "${BACKUP_DIR}/schemas_${TIMESTAMP}.tar.gz" gs://echo-backups/
#     # aws s3 cp "${BACKUP_DIR}/schemas_${TIMESTAMP}.tar.gz" s3://echo-backups/
# fi

echo "[SUCCESS] Backup process completed!"
