-- =====================================================
-- Initial Schema Migration
-- Description: Creates all base schemas for Echo Backend
-- =====================================================

-- Enable Required Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- This migration serves as a baseline.
-- The actual schema creation is handled by the individual schema files.
-- This ensures we have a migration baseline for future incremental migrations.

-- Track migration version
CREATE TABLE IF NOT EXISTS schema_migrations (
    version BIGINT PRIMARY KEY,
    description TEXT,
    applied_at TIMESTAMPTZ DEFAULT NOW()
);

INSERT INTO schema_migrations (version, description)
VALUES (1, 'Initial schema baseline')
ON CONFLICT (version) DO NOTHING;
