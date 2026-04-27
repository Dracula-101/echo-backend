CREATE OR REPLACE FUNCTION messages.get_conversation_messages(
    p_conversation_id UUID, p_user_id UUID,
    p_limit INTEGER DEFAULT 50, p_before_message_id UUID DEFAULT NULL
) RETURNS TABLE (
    message_id      UUID, sender_id UUID, sender_username VARCHAR,
    sender_avatar   TEXT, content TEXT, message_type VARCHAR,
    created_at      TIMESTAMPTZ, is_edited BOOLEAN, reactions JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        m.id,
        m.sender_user_id,
        p.username,
        p.avatar_url,
        CASE WHEN m.is_deleted THEN '[Message deleted]' ELSE m.content END,
        m.message_type,
        m.created_at,
        m.is_edited,
        COALESCE((
            SELECT jsonb_agg(
                jsonb_build_object(
                    'user_id', r.user_id, 'reaction_type', r.reaction_type, 'created_at', r.created_at
                ) ORDER BY r.created_at
            )
            FROM messages.reactions r WHERE r.message_id = m.id
        ), '[]'::JSONB)
    FROM messages.messages m
    JOIN users.profiles p ON p.user_id = m.sender_user_id
    WHERE m.conversation_id = p_conversation_id
      AND (p_before_message_id IS NULL OR m.id < p_before_message_id)
      AND NOT EXISTS (
          SELECT 1 FROM users.blocked_users b
          WHERE (b.user_id = p_user_id AND b.blocked_user_id = m.sender_user_id)
             OR (b.user_id = m.sender_user_id AND b.blocked_user_id = p_user_id)
      )
    ORDER BY m.created_at DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION messages.delete_message(
    p_message_id UUID, p_user_id UUID, p_delete_for VARCHAR DEFAULT 'sender'
) RETURNS VOID AS $$
BEGIN
    IF p_delete_for = 'everyone' THEN
        UPDATE messages.messages
        SET is_deleted = TRUE, deleted_at = NOW(), deleted_for = 'everyone', content = '[Message deleted]'
        WHERE id = p_message_id AND sender_user_id = p_user_id AND is_deleted = FALSE;
    ELSE
        UPDATE messages.messages
        SET is_deleted = TRUE, deleted_at = NOW(), deleted_for = 'sender'
        WHERE id = p_message_id AND sender_user_id = p_user_id AND is_deleted = FALSE;
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION messages.cleanup_expired_typing_indicators() RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    DELETE FROM messages.typing_indicators WHERE expires_at < NOW();
    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION messages.expire_disappearing_messages() RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    UPDATE messages.messages
    SET is_deleted = TRUE, deleted_at = NOW(), deleted_for = 'everyone', content = '[Message expired]'
    WHERE expires_at < NOW() AND is_deleted = FALSE;
    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ----------------------------------------------------------------------
-- messages.mark_messages_as_read(user_id, conversation_id, message_id)
-- ----------------------------------------------------------------------
-- Atomic read-receipt update. For every message in the conversation up
-- to and including the cursor message:
--   1. Flip the caller's delivery_status row to 'read' (if not already).
--      The update_message_delivery_counts trigger fires per row and
--      recalculates messages.read_count, so this function does not
--      update read_count directly.
--   2. Reset the caller's unread_count and mention_count to 0 and
--      advance last_read_message_id / last_read_at. The caller is
--      expected to pass the latest visible message ID — the spec's
--      semantics zero unread_count unconditionally rather than computing
--      a remainder, since the client typically reads-to-bottom.
--
-- UUIDs from gen_random_uuid() are not time-ordered, so the cursor's
-- created_at is resolved first and ordering is done on that, not on id.
--
-- Returns: number of delivery_status rows that transitioned to 'read'.
-- ----------------------------------------------------------------------
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
