-- =====================================================
-- Rollback Presence Schema
-- =====================================================

-- Drop triggers
DROP TRIGGER IF EXISTS trigger_update_connection_count ON presence.connections;

-- Drop functions
DROP FUNCTION IF EXISTS presence.update_connection_count();
DROP FUNCTION IF EXISTS presence.cleanup_stale_connections();
DROP FUNCTION IF EXISTS presence.cleanup_expired_typing();
DROP FUNCTION IF EXISTS presence.update_user_status(UUID, VARCHAR, VARCHAR);

-- Drop tables
DROP TABLE IF EXISTS presence.status_history;
DROP TABLE IF EXISTS presence.subscriptions;
DROP TABLE IF EXISTS presence.typing_indicators;
DROP TABLE IF EXISTS presence.activity_log;
DROP TABLE IF EXISTS presence.connections;
DROP TABLE IF EXISTS presence.user_status;

-- Drop schema
DROP SCHEMA IF EXISTS presence CASCADE;

-- Remove migration tracking
DELETE FROM schema_migrations WHERE version = 2;
