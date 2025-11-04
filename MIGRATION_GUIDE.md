# Database Migration System - Implementation Guide

## What's New

I've set up a proper database migration system for your Echo Backend messaging service and created comprehensive documentation.

### Files Created

1. **Migration Files:**
   - `migrations/postgres/000001_init_schema.up.sql` - Baseline migration
   - `migrations/postgres/000001_init_schema.down.sql` - Rollback baseline
   - `migrations/postgres/000002_add_presence_schema.up.sql` - NEW: Presence tracking schema
   - `migrations/postgres/000002_add_presence_schema.down.sql` - Rollback presence schema

2. **Migration Runner:**
   - `infra/scripts/run-migrations.sh` - Script to apply/rollback migrations

3. **Documentation:**
   - `QUICKSTART.md` - Complete quickstart guide with architecture overview
   - `MIGRATION_GUIDE.md` - This file

4. **Makefile Updates:**
   - Added `make db-migrate` - Apply migrations
   - Added `make db-migrate-down` - Rollback migrations
   - Added `make db-migrate-status` - Check migration status

---

## New Presence Schema

I've created a dedicated **`presence`** schema for your presence-service with the following tables:

### Tables

1. **`presence.user_status`** - Current online/offline status
   - Real-time presence tracking
   - Custom status messages
   - Visibility settings
   - Multi-device support

2. **`presence.connections`** - Active WebSocket/connection sessions
   - Per-device connection tracking
   - Heartbeat monitoring
   - Server routing for distributed systems

3. **`presence.activity_log`** - User activity tracking
   - Typing, viewing, recording activities
   - Context tracking (conversation, profile, etc.)
   - Duration tracking

4. **`presence.typing_indicators`** - Real-time typing status
   - Per-conversation typing state
   - Auto-expiring (10 seconds TTL)

5. **`presence.subscriptions`** - Presence update subscriptions
   - Who is subscribed to whose presence
   - Contact-based, conversation-based, or temporary

6. **`presence.status_history`** - Historical presence data
   - For analytics and debugging
   - Tracks all status changes

### Helper Functions

```sql
-- Update user status
SELECT presence.update_user_status('user_id', 'online', 'device_id');

-- Clean up expired typing indicators
SELECT presence.cleanup_expired_typing();

-- Mark stale connections as disconnected (30s timeout)
SELECT presence.cleanup_stale_connections();
```

---

## How to Use

### Option 1: Fresh Database (Recommended for New Setups)

If you're starting fresh or want to reset everything:

```bash
# 1. Start services
make up

# 2. Initialize database with all schemas
make db-init

# 3. Apply migrations (including presence schema)
make db-migrate

# 4. Seed test data
make db-seed

# 5. Verify
make db-migrate-status
```

### Option 2: Existing Database (Add Presence Schema Only)

If you already have data and just want to add the presence schema:

```bash
# 1. Check current migration status
make db-migrate-status

# 2. Apply new migrations (will only apply unapplied ones)
make db-migrate

# This will add the presence schema while keeping existing data intact
```

### Option 3: Manual Schema Load (Alternative)

If you prefer not to use migrations yet:

```bash
# Connect to database
make db-connect

# Manually load presence schema
\i /database/schemas/presence-schema.sql  # If you create it

# Or apply the migration file directly
\i /migrations/postgres/000002_add_presence_schema.up.sql
```

---

## Migration Commands

### Apply Migrations

```bash
# Apply all pending migrations
make db-migrate

# Or use the script directly
cd infra/scripts
./run-migrations.sh up
```

### Check Status

```bash
# Show which migrations are applied
make db-migrate-status

# Example output:
# ✓ 000001_init_schema.up.sql (applied)
# ✓ 000002_add_presence_schema.up.sql (applied)
# ○ 000003_add_indexes.up.sql (pending)
```

### Rollback

```bash
# Rollback the last migration
make db-migrate-down

# Or rollback all migrations to a specific version
cd infra/scripts
./run-migrations.sh down 1  # Rollback to version 1
```

### Force Re-apply

```bash
# Force re-apply a specific migration (useful for testing)
cd infra/scripts
./run-migrations.sh force 2
```

---

## Creating New Migrations

### Step 1: Create Migration Files

```bash
# Navigate to migrations directory
cd migrations/postgres

# Create up migration
cat > 000003_your_migration_name.up.sql << 'EOF'
-- Your SQL changes here
ALTER TABLE users.profiles ADD COLUMN last_active_at TIMESTAMPTZ;

-- Track the migration
INSERT INTO schema_migrations (version, description)
VALUES (3, 'Add last_active_at to profiles')
ON CONFLICT (version) DO NOTHING;
EOF

# Create down migration (rollback)
cat > 000003_your_migration_name.down.sql << 'EOF'
-- Reverse your changes
ALTER TABLE users.profiles DROP COLUMN last_active_at;

-- Remove migration tracking
DELETE FROM schema_migrations WHERE version = 3;
EOF
```

### Step 2: Test Migration

```bash
# Apply the migration
make db-migrate

# Verify it worked
make db-connect
# Run \d users.profiles to see the new column

# Test rollback
make db-migrate-down

# Verify rollback worked
make db-connect
# Run \d users.profiles to confirm column is gone

# Re-apply
make db-migrate
```

---

## Database Schema Overview

Your messaging backend now has **8 schemas**:

| Schema | Tables | Purpose | Status |
|--------|--------|---------|--------|
| `auth` | 6 | Authentication & sessions | ✅ Existing |
| `users` | 4 | User profiles & contacts | ✅ Existing |
| `messages` | 20 | Conversations & messaging | ✅ Existing |
| `media` | 2 | File storage & metadata | ✅ Existing |
| `notifications` | 2 | Push notifications | ✅ Existing |
| `analytics` | 6 | User analytics & events | ✅ Existing |
| `location` | 2 | IP geolocation data | ✅ Existing |
| **`presence`** | 6 | **Real-time presence** | 🆕 **NEW!** |

---

## Testing the Presence Schema

### 1. Verify Schema Creation

```bash
# Connect to database
make db-connect

# Check presence tables
\dt presence.*

# Expected output:
# presence.user_status
# presence.connections
# presence.activity_log
# presence.typing_indicators
# presence.subscriptions
# presence.status_history
```

### 2. Insert Test Data

```sql
-- Set user status
SELECT presence.update_user_status(
    'a1111111-1111-1111-1111-111111111111',
    'online',
    'device-123'
);

-- Verify
SELECT * FROM presence.user_status
WHERE user_id = 'a1111111-1111-1111-1111-111111111111';
```

### 3. Test Cleanup Functions

```sql
-- Clean up expired typing indicators
SELECT presence.cleanup_expired_typing();

-- Mark stale connections
SELECT presence.cleanup_stale_connections();
```

---

## Integration with Presence Service

Once you implement the presence-service, you can use this schema:

### Example Go Code

```go
// Update user status when they connect
func (s *PresenceService) UpdateStatus(ctx context.Context, req *pb.UpdateStatusRequest) error {
    query := `
        SELECT presence.update_user_status($1, $2, $3)
    `
    _, err := s.db.ExecContext(ctx, query, req.UserId, req.Status, req.DeviceId)
    return err
}

// Track active connection
func (s *PresenceService) TrackConnection(ctx context.Context, conn *Connection) error {
    query := `
        INSERT INTO presence.connections (
            user_id, connection_id, device_id, device_type,
            ip_address, user_agent, server_id, last_heartbeat_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
        ON CONFLICT (connection_id) DO UPDATE SET
            last_heartbeat_at = NOW()
    `
    _, err := s.db.ExecContext(ctx, query,
        conn.UserId, conn.ConnectionId, conn.DeviceId, conn.DeviceType,
        conn.IpAddress, conn.UserAgent, conn.ServerId,
    )
    return err
}

// Set typing indicator
func (s *PresenceService) SetTyping(ctx context.Context, req *pb.TypingRequest) error {
    query := `
        INSERT INTO presence.typing_indicators (
            conversation_id, user_id, device_id, expires_at
        ) VALUES ($1, $2, $3, NOW() + INTERVAL '10 seconds')
        ON CONFLICT (conversation_id, user_id) DO UPDATE SET
            last_updated_at = NOW(),
            expires_at = NOW() + INTERVAL '10 seconds'
    `
    _, err := s.db.ExecContext(ctx, query,
        req.ConversationId, req.UserId, req.DeviceId,
    )
    return err
}
```

---

## Benefits of This Migration System

1. **Version Control:** Track all database changes in git
2. **Incremental Updates:** Apply changes without rebuilding entire database
3. **Rollback Support:** Easily revert problematic changes
4. **Production Ready:** Safe to use in production environments
5. **Team Friendly:** Multiple developers can create migrations independently
6. **Audit Trail:** `schema_migrations` table tracks what's been applied

---

## Next Steps

### 1. Implement Presence Service

The schema is ready! Now you can:
- Create WebSocket handlers for real-time connections
- Implement presence status updates
- Add typing indicator broadcasting
- Build presence subscription system

### 2. Add More Migrations

Consider creating migrations for:
- Additional indexes for performance
- Full-text search setup for messages
- Partitioning for large tables (messages, analytics)
- Database functions for complex queries

### 3. Setup Automated Migrations

In production, integrate migrations into your deployment pipeline:

```bash
# In CI/CD pipeline
./infra/scripts/run-migrations.sh up

# Then deploy new code
docker-compose -f docker-compose.prod.yml up -d
```

---

## Troubleshooting

### Migration Failed

```bash
# Check what went wrong
make db-migrate-status

# View PostgreSQL logs
docker logs echo-postgres

# Manually fix the issue in the database
make db-connect

# Force re-apply the migration
cd infra/scripts
./run-migrations.sh force <version>
```

### Rollback Not Working

```bash
# Check if down migration file exists
ls migrations/postgres/*down.sql

# Manually apply rollback
make db-connect
\i migrations/postgres/000002_add_presence_schema.down.sql
```

### Schema Conflicts

If you get "table already exists" errors:

```bash
# Check what's currently in the database
make db-connect
\dt presence.*

# Either drop manually or use rollback
DROP SCHEMA IF EXISTS presence CASCADE;

# Then re-apply
make db-migrate
```

---

## Questions?

- Check the `QUICKSTART.md` for overall architecture
- See `database/schemas/` for full schema definitions
- Review `infra/scripts/init-db.sh` for initialization process

Happy coding! 🚀
