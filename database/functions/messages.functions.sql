-- =====================================================
-- MESSAGES SCHEMA — UTILITY FUNCTIONS
-- =====================================================
--
-- Description:  Callable utility functions for the
--               messages schema. These are invoked by
--               application code or other SQL functions
--               — NOT by triggers.
--
-- Note:         Trigger handler functions (RETURNS TRIGGER)
--               live in messages.trigger_functions.sql.
--
-- Dependencies: messages schema tables must exist.
--               users.profiles (for get_conversation_messages).
--               users.blocked_users (for get_conversation_messages).
--
-- =====================================================


-- -------------------------------------------------
-- messages.mark_messages_as_read(...)
-- -------------------------------------------------
-- Marks all messages in a conversation up to (and
-- including) p_message_id as 'read' for p_user_id.
-- Updates delivery_status rows, resets the participant's
-- unread_count and mention_count to zero, and
-- recalculates read_count on affected messages.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.mark_messages_as_read(
    p_user_id UUID,
    p_conversation_id UUID,
    p_message_id UUID
)
RETURNS VOID AS $$
BEGIN
    -- Update delivery status for all unread messages up to this point
    UPDATE messages.delivery_status
    SET status = 'read',
        read_at = NOW()
    WHERE user_id = p_user_id
      AND message_id IN (
          SELECT id FROM messages.messages
          WHERE conversation_id = p_conversation_id
            AND id <= p_message_id
            AND status != 'read'
      );

    -- Reset participant counters and record last-read position
    UPDATE messages.conversation_participants
    SET last_read_message_id = p_message_id,
        last_read_at = NOW(),
        unread_count = 0,
        mention_count = 0
    WHERE user_id = p_user_id
      AND conversation_id = p_conversation_id;

    -- Recalculate read_count on each affected message
    UPDATE messages.messages
    SET read_count = (
        SELECT COUNT(*) FROM messages.delivery_status
        WHERE message_id = messages.messages.id
          AND status = 'read'
    )
    WHERE conversation_id = p_conversation_id
      AND id <= p_message_id;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- messages.delete_message(...)
-- -------------------------------------------------
-- Soft-deletes a message. Supports two modes:
--   'everyone' — replaces content with placeholder,
--                visible to all participants
--   'sender'   — marks deleted only for the sender
--
-- Only the original sender may delete a message.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.delete_message(
    p_message_id UUID,
    p_user_id UUID,
    p_delete_for VARCHAR DEFAULT 'sender'
)
RETURNS VOID AS $$
BEGIN
    IF p_delete_for = 'everyone' THEN
        UPDATE messages.messages
        SET is_deleted = TRUE,
            deleted_at = NOW(),
            deleted_for = 'everyone',
            content = '[Message deleted]'
        WHERE id = p_message_id
          AND sender_user_id = p_user_id;
    ELSE
        UPDATE messages.messages
        SET is_deleted = TRUE,
            deleted_at = NOW(),
            deleted_for = 'sender'
        WHERE id = p_message_id
          AND sender_user_id = p_user_id;
    END IF;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- messages.cleanup_expired_typing_indicators()
-- -------------------------------------------------
-- Deletes all typing indicator rows whose expires_at
-- has passed. Intended to be called by a periodic
-- cleanup job (the trigger-based cleanup handles
-- opportunistic cleanup on INSERT).
--
-- Returns the number of rows deleted.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.cleanup_expired_typing_indicators()
RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    DELETE FROM messages.typing_indicators
    WHERE expires_at < NOW();

    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- messages.expire_disappearing_messages()
-- -------------------------------------------------
-- Soft-deletes all messages whose expires_at has
-- passed. Replaces content with '[Message expired]'.
-- Intended to be called by a periodic cleanup job.
--
-- Returns the number of messages expired.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.expire_disappearing_messages()
RETURNS INTEGER AS $$
DECLARE
    v_deleted_count INTEGER;
BEGIN
    UPDATE messages.messages
    SET is_deleted = TRUE,
        deleted_at = NOW(),
        deleted_for = 'everyone',
        content = '[Message expired]'
    WHERE expires_at < NOW()
      AND is_deleted = FALSE;

    GET DIAGNOSTICS v_deleted_count = ROW_COUNT;
    RETURN v_deleted_count;
END;
$$ LANGUAGE plpgsql;


-- -------------------------------------------------
-- messages.get_conversation_messages(...)
-- -------------------------------------------------
-- Returns paginated messages for a conversation,
-- excluding messages from blocked users. Supports
-- cursor-based pagination via p_before_message_id.
--
-- Reactions are aggregated into a JSONB array per
-- message. Deleted messages show placeholder content.
-- -------------------------------------------------
CREATE OR REPLACE FUNCTION messages.get_conversation_messages(
    p_conversation_id UUID,
    p_user_id UUID,
    p_limit INTEGER DEFAULT 50,
    p_before_message_id UUID DEFAULT NULL
)
RETURNS TABLE (
    message_id UUID,
    sender_id UUID,
    sender_username VARCHAR,
    sender_avatar TEXT,
    content TEXT,
    message_type VARCHAR,
    created_at TIMESTAMPTZ,
    is_edited BOOLEAN,
    reactions JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        m.id,
        m.sender_user_id,
        p.username,
        p.avatar_url,
        CASE
            WHEN m.is_deleted THEN '[Message deleted]'
            ELSE m.content
        END,
        m.message_type,
        m.created_at,
        m.is_edited,
        COALESCE(
            (SELECT jsonb_agg(
                jsonb_build_object(
                    'user_id', r.user_id,
                    'reaction_type', r.reaction_type,
                    'created_at', r.created_at
                )
            )
            FROM messages.reactions r
            WHERE r.message_id = m.id),
            '[]'::jsonb
        )
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
