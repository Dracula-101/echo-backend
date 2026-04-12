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
