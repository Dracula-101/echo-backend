-- =====================================================
-- Rollback: Remove messages.outbox
-- =====================================================

DROP INDEX IF EXISTS messages.idx_messages_outbox_aggregate;
DROP INDEX IF EXISTS messages.idx_messages_outbox_event_type;
DROP INDEX IF EXISTS messages.idx_messages_outbox_unprocessed;

DROP TABLE IF EXISTS messages.outbox;

DELETE FROM schema_migrations WHERE version = 3;
