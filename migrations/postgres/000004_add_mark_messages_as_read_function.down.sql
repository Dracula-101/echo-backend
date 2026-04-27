-- =====================================================
-- Rollback: Remove messages.mark_messages_as_read() function
-- =====================================================

DROP FUNCTION IF EXISTS messages.mark_messages_as_read(UUID, UUID, UUID);

DELETE FROM schema_migrations WHERE version = 4;
