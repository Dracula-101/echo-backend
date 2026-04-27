-- =====================================================
-- Migration: Add messages.mark_messages_as_read() function
-- =====================================================
-- Atomic read-receipt update used by the POST /{id}/read endpoint and
-- by inline read-marking on history fetch. Flips delivery_status to
-- 'read' for every message up to and including the cursor message,
-- then zeroes the caller's unread/mention counters.

CREATE OR REPLACE FUNCTION messages.mark_messages_as_read(
    p_user_id UUID,
    p_conversation_id UUID,
    p_message_id UUID
) RETURNS INTEGER AS $$
DECLARE
    v_cursor_created_at TIMESTAMPTZ;
    v_updated_count     INTEGER;
BEGIN
    SELECT created_at INTO v_cursor_created_at
    FROM messages.messages
    WHERE id = p_message_id
      AND conversation_id = p_conversation_id;

    IF v_cursor_created_at IS NULL THEN
        RAISE EXCEPTION 'Message % not found in conversation %',
            p_message_id, p_conversation_id;
    END IF;

    UPDATE messages.delivery_status ds
    SET status     = 'read',
        read_at    = NOW(),
        updated_at = NOW()
    FROM messages.messages m
    WHERE ds.message_id     = m.id
      AND ds.user_id        = p_user_id
      AND ds.status        != 'read'
      AND m.conversation_id = p_conversation_id
      AND m.created_at     <= v_cursor_created_at;

    GET DIAGNOSTICS v_updated_count = ROW_COUNT;

    UPDATE messages.conversation_participants
    SET last_read_message_id = p_message_id,
        last_read_at         = NOW(),
        unread_count         = 0,
        mention_count        = 0,
        updated_at           = NOW()
    WHERE user_id         = p_user_id
      AND conversation_id = p_conversation_id;

    RETURN v_updated_count;
END;
$$ LANGUAGE plpgsql;

INSERT INTO schema_migrations (version, description)
VALUES (4, 'Add messages.mark_messages_as_read() function')
ON CONFLICT (version) DO NOTHING;
