-- =====================================================
-- MESSAGES SCHEMA - FUNCTIONS
-- =====================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION messages.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to update conversation last_message info
CREATE OR REPLACE FUNCTION messages.update_conversation_last_message()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messages.conversations
    SET last_message_id = NEW.id,
        last_message_at = NEW.created_at,
        last_activity_at = NEW.created_at,
        message_count = message_count + 1
    WHERE id = NEW.conversation_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to increment unread count for participants
CREATE OR REPLACE FUNCTION messages.increment_unread_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messages.conversation_participants
    SET unread_count = unread_count + 1
    WHERE conversation_id = NEW.conversation_id
    AND user_id != NEW.sender_user_id
    AND left_at IS NULL
    AND removed_at IS NULL;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to increment mention count
CREATE OR REPLACE FUNCTION messages.increment_mention_count()
RETURNS TRIGGER AS $$
DECLARE
    v_mentioned_user_id UUID;
BEGIN
    IF NEW.mentions IS NOT NULL AND jsonb_array_length(NEW.mentions) > 0 THEN
        FOR v_mentioned_user_id IN 
            SELECT (mention->>'user_id')::UUID 
            FROM jsonb_array_elements(NEW.mentions) AS mention
        LOOP
            UPDATE messages.conversation_participants
            SET mention_count = mention_count + 1
            WHERE conversation_id = NEW.conversation_id
            AND user_id = v_mentioned_user_id;
        END LOOP;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to mark messages as read
CREATE OR REPLACE FUNCTION messages.mark_messages_as_read(
    p_user_id UUID,
    p_conversation_id UUID,
    p_message_id UUID
)
RETURNS VOID AS $$
BEGIN
    -- Update delivery status
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
    
    -- Update participant info
    UPDATE messages.conversation_participants
    SET last_read_message_id = p_message_id,
        last_read_at = NOW(),
        unread_count = 0,
        mention_count = 0
    WHERE user_id = p_user_id
    AND conversation_id = p_conversation_id;
    
    -- Update message read count
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

-- Function to delete message
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
        -- For delete for self, we'd need a separate table to track individual deletions
        -- This is a simplified version
        UPDATE messages.messages
        SET is_deleted = TRUE,
            deleted_at = NOW(),
            deleted_for = 'sender'
        WHERE id = p_message_id
        AND sender_user_id = p_user_id;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to update message search index
CREATE OR REPLACE FUNCTION messages.update_search_index()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO messages.search_index (
        message_id, conversation_id, user_id, content_tsvector
    ) VALUES (
        NEW.id, NEW.conversation_id, NEW.sender_user_id,
        to_tsvector('english', COALESCE(NEW.content, ''))
    )
    ON CONFLICT (message_id) DO UPDATE SET
        content_tsvector = to_tsvector('english', COALESCE(NEW.content, '')),
        updated_at = NOW();
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up expired typing indicators
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

-- Function to update poll vote counts
CREATE OR REPLACE FUNCTION messages.update_poll_votes()
RETURNS TRIGGER AS $$
BEGIN
    -- Update option vote count
    UPDATE messages.poll_options
    SET vote_count = (
        SELECT COUNT(*) FROM messages.poll_votes
        WHERE poll_option_id = NEW.poll_option_id
    )
    WHERE id = NEW.poll_option_id;
    
    -- Update total poll votes
    UPDATE messages.polls
    SET total_votes = (
        SELECT COUNT(*) FROM messages.poll_votes
        WHERE poll_id = NEW.poll_id
    )
    WHERE id = NEW.poll_id;
    
    -- Update vote percentages
    UPDATE messages.poll_options po
    SET vote_percentage = CASE 
        WHEN p.total_votes > 0 THEN 
            (po.vote_count::DECIMAL / p.total_votes * 100)
        ELSE 0
    END
    FROM messages.polls p
    WHERE po.poll_id = p.id
    AND p.id = NEW.poll_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to update message reaction count
CREATE OR REPLACE FUNCTION messages.update_reaction_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messages.messages
    SET reaction_count = (
        SELECT COUNT(*) FROM messages.reactions
        WHERE message_id = COALESCE(NEW.message_id, OLD.message_id)
    )
    WHERE id = COALESCE(NEW.message_id, OLD.message_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Function to update conversation member count
CREATE OR REPLACE FUNCTION messages.update_member_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE messages.conversations
    SET member_count = (
        SELECT COUNT(*)
        FROM messages.conversation_participants
        WHERE conversation_id = COALESCE(NEW.conversation_id, OLD.conversation_id)
        AND left_at IS NULL
        AND removed_at IS NULL
    )
    WHERE id = COALESCE(NEW.conversation_id, OLD.conversation_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Function to create default conversation settings
CREATE OR REPLACE FUNCTION messages.create_default_conversation_settings()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO messages.conversation_settings (conversation_id)
    VALUES (NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to expire disappearing messages
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

-- Function to get conversation messages
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