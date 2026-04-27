-- =====================================================
-- Migration: Add messages.outbox for transactional Kafka publishing
-- =====================================================
-- The relay process polls unprocessed rows, publishes them to Kafka,
-- and then sets processed_at. Writing the row inside the same business
-- transaction guarantees at-least-once delivery without dual-write
-- inconsistency.

CREATE TABLE IF NOT EXISTS messages.outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    topic VARCHAR(255) NOT NULL,
    aggregate_id UUID,
    payload JSONB NOT NULL,
    user_id UUID,
    headers JSONB DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    processing_attempts INTEGER DEFAULT 0,
    last_error TEXT
);

CREATE INDEX IF NOT EXISTS idx_messages_outbox_unprocessed ON messages.outbox(created_at)
    WHERE processed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_messages_outbox_event_type ON messages.outbox(event_type);
CREATE INDEX IF NOT EXISTS idx_messages_outbox_aggregate ON messages.outbox(aggregate_id) WHERE aggregate_id IS NOT NULL;

INSERT INTO schema_migrations (version, description)
VALUES (3, 'Add messages.outbox for transactional Kafka publishing')
ON CONFLICT (version) DO NOTHING;
