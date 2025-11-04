-- =====================================================
-- Rollback Initial Schema
-- =====================================================

-- Drop all schemas
DROP SCHEMA IF EXISTS location CASCADE;
DROP SCHEMA IF EXISTS analytics CASCADE;
DROP SCHEMA IF EXISTS notifications CASCADE;
DROP SCHEMA IF EXISTS media CASCADE;
DROP SCHEMA IF EXISTS messages CASCADE;
DROP SCHEMA IF EXISTS users CASCADE;
DROP SCHEMA IF EXISTS auth CASCADE;

-- Remove migration tracking
DROP TABLE IF EXISTS schema_migrations;
