-- =====================================================
-- MESSAGES SCHEMA - TRIGGERS
-- =====================================================

-- Trigger to update updated_at on messages.conversations
CREATE TRIGGER trigger_messages_conversations_updated_at
    BEFORE UPDATE ON messages.conversations
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();

-- Trigger to update updated_at on messages.conversation_participants
CREATE TRIGGER trigger_messages_participants_updated_at
    BEFORE UPDATE ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();

-- Trigger to update updated_at on messages.messages
CREATE TRIGGER trigger_messages_updated_at
    BEFORE UPDATE ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();

-- Trigger to update updated_at on messages.message_reports
CREATE TRIGGER trigger_messages_reports_updated_at
    BEFORE UPDATE ON messages.message_reports
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();

-- Trigger to update updated_at on messages.drafts
CREATE TRIGGER trigger_messages_drafts_updated_at
    BEFORE UPDATE ON messages.drafts
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();

-- Trigger to update conversation info when message is created
CREATE TRIGGER trigger_messages_update_conversation
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_conversation_last_message();

-- Trigger to increment unread counts
CREATE TRIGGER trigger_messages_increment_unread
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.increment_unread_count();

-- Trigger to increment mention counts
CREATE TRIGGER trigger_messages_increment_mentions
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    WHEN (NEW.mentions IS NOT NULL AND jsonb_array_length(NEW.mentions) > 0)
    EXECUTE FUNCTION messages.increment_mention_count();

-- Trigger to update search index
CREATE TRIGGER trigger_messages_update_search_index
    AFTER INSERT OR UPDATE ON messages.messages
    FOR EACH ROW
    WHEN (NEW.message_type = 'text' AND NEW.content IS NOT NULL AND NOT NEW.is_deleted)
    EXECUTE FUNCTION messages.update_search_index();

-- Trigger to update reaction counts
CREATE TRIGGER trigger_reactions_update_count_insert
    AFTER INSERT ON messages.reactions
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_reaction_count();

CREATE TRIGGER trigger_reactions_update_count_delete
    AFTER DELETE ON messages.reactions
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_reaction_count();

-- Trigger to update member count
CREATE TRIGGER trigger_participants_update_member_count_insert
    AFTER INSERT ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_member_count();

CREATE TRIGGER trigger_participants_update_member_count_update
    AFTER UPDATE ON messages.conversation_participants
    FOR EACH ROW
    WHEN (OLD.left_at IS DISTINCT FROM NEW.left_at OR OLD.removed_at IS DISTINCT FROM NEW.removed_at)
    EXECUTE FUNCTION messages.update_member_count();

CREATE TRIGGER trigger_participants_update_member_count_delete
    AFTER DELETE ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_member_count();

-- Trigger to create default conversation settings
CREATE TRIGGER trigger_conversations_create_settings
    AFTER INSERT ON messages.conversations
    FOR EACH ROW
    EXECUTE FUNCTION messages.create_default_conversation_settings();

-- Trigger to update poll vote counts
CREATE TRIGGER trigger_poll_votes_update_counts
    AFTER INSERT ON messages.poll_votes
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_poll_votes();

CREATE TRIGGER trigger_poll_votes_update_counts_delete
    AFTER DELETE ON messages.poll_votes
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_poll_votes();

-- Trigger to set message edited timestamp
CREATE OR REPLACE FUNCTION messages.set_edited_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.content IS DISTINCT FROM NEW.content AND NEW.content IS NOT NULL THEN
        NEW.is_edited = TRUE;
        NEW.edited_at = NOW();
        
        -- Store edit history
        NEW.edit_history = COALESCE(NEW.edit_history, '[]'::JSONB) || 
            jsonb_build_object(
                'edited_at', NOW(),
                'previous_content', OLD.content
            );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_messages_set_edited
    BEFORE UPDATE ON messages.messages
    FOR EACH ROW
    WHEN (OLD.content IS DISTINCT FROM NEW.content)
    EXECUTE FUNCTION messages.set_edited_timestamp();

-- Trigger to update delivery status counts
CREATE OR REPLACE FUNCTION messages.update_message_delivery_counts()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR (TG_OP = 'UPDATE' AND OLD.status IS DISTINCT FROM NEW.status) THEN
        UPDATE messages.messages
        SET delivery_count = (
            SELECT COUNT(*) FROM messages.delivery_status
            WHERE message_id = COALESCE(NEW.message_id, OLD.message_id)
            AND status IN ('delivered', 'read')
        ),
        read_count = (
            SELECT COUNT(*) FROM messages.delivery_status
            WHERE message_id = COALESCE(NEW.message_id, OLD.message_id)
            AND status = 'read'
        )
        WHERE id = COALESCE(NEW.message_id, OLD.message_id);
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_delivery_status_update_counts
    AFTER INSERT OR UPDATE ON messages.delivery_status
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_message_delivery_counts();

-- Trigger to create delivery status for all participants on message insert
CREATE OR REPLACE FUNCTION messages.create_delivery_status()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO messages.delivery_status (message_id, user_id, status)
    SELECT NEW.id, cp.user_id, 'sent'
    FROM messages.conversation_participants cp
    WHERE cp.conversation_id = NEW.conversation_id
    AND cp.user_id != NEW.sender_user_id
    AND cp.left_at IS NULL
    AND cp.removed_at IS NULL;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_messages_create_delivery_status
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.create_delivery_status();

-- Trigger to update reply count
CREATE OR REPLACE FUNCTION messages.update_reply_count()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.parent_message_id IS NOT NULL THEN
        UPDATE messages.messages
        SET reply_count = reply_count + 1,
            last_reply_at = NEW.created_at
        WHERE id = NEW.parent_message_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_messages_update_reply_count
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    WHEN (NEW.parent_message_id IS NOT NULL)
    EXECUTE FUNCTION messages.update_reply_count();

-- Trigger to prevent sending messages in left/removed conversations
CREATE OR REPLACE FUNCTION messages.validate_participant_can_send()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM messages.conversation_participants
        WHERE conversation_id = NEW.conversation_id
        AND user_id = NEW.sender_user_id
        AND left_at IS NULL
        AND removed_at IS NULL
        AND can_send_messages = TRUE
    ) THEN
        RAISE EXCEPTION 'User % cannot send messages in conversation %', NEW.sender_user_id, NEW.conversation_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_messages_validate_sender
    BEFORE INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.validate_participant_can_send();

-- Trigger to prevent blocked users from messaging each other
CREATE OR REPLACE FUNCTION messages.validate_not_blocked()
RETURNS TRIGGER AS $$
DECLARE
    v_recipient_id UUID;
BEGIN
    -- For direct conversations, check if users have blocked each other
    IF EXISTS (
        SELECT 1 FROM messages.conversations c
        WHERE c.id = NEW.conversation_id
        AND c.conversation_type = 'direct'
    ) THEN
        -- Get the other participant
        SELECT user_id INTO v_recipient_id
        FROM messages.conversation_participants
        WHERE conversation_id = NEW.conversation_id
        AND user_id != NEW.sender_user_id
        LIMIT 1;
        
        IF v_recipient_id IS NOT NULL THEN
            IF users.is_blocked(NEW.sender_user_id, v_recipient_id) 
               OR users.is_blocked(v_recipient_id, NEW.sender_user_id) THEN
                RAISE EXCEPTION 'Cannot send message to blocked user';
            END IF;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_messages_validate_not_blocked
    BEFORE INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.validate_not_blocked();

-- Trigger to auto-close polls
CREATE OR REPLACE FUNCTION messages.auto_close_poll()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.closes_at IS NOT NULL AND NEW.closes_at <= NOW() AND NOT NEW.is_closed THEN
        NEW.is_closed = TRUE;
        NEW.closed_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_polls_auto_close
    BEFORE UPDATE ON messages.polls
    FOR EACH ROW
    WHEN (NEW.closes_at IS NOT NULL AND NEW.closes_at <= NOW())
    EXECUTE FUNCTION messages.auto_close_poll();

-- Trigger to prevent voting on closed polls
CREATE OR REPLACE FUNCTION messages.validate_poll_not_closed()
RETURNS TRIGGER AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM messages.polls
        WHERE id = NEW.poll_id
        AND is_closed = TRUE
    ) THEN
        RAISE EXCEPTION 'Cannot vote on closed poll';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_poll_votes_validate_not_closed
    BEFORE INSERT ON messages.poll_votes
    FOR EACH ROW
    EXECUTE FUNCTION messages.validate_poll_not_closed();

-- Trigger to prevent multiple votes if not allowed
CREATE OR REPLACE FUNCTION messages.validate_single_vote()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM messages.polls
        WHERE id = NEW.poll_id
        AND allow_multiple_answers = TRUE
    ) THEN
        IF EXISTS (
            SELECT 1 FROM messages.poll_votes
            WHERE poll_id = NEW.poll_id
            AND user_id = NEW.user_id
        ) THEN
            RAISE EXCEPTION 'User has already voted on this poll';
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_poll_votes_validate_single
    BEFORE INSERT ON messages.poll_votes
    FOR EACH ROW
    EXECUTE FUNCTION messages.validate_single_vote();

-- Trigger to delete expired typing indicators
CREATE OR REPLACE FUNCTION messages.cleanup_typing_indicator()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM messages.typing_indicators
    WHERE expires_at < NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_typing_indicators_cleanup
    BEFORE INSERT ON messages.typing_indicators
    FOR EACH ROW
    EXECUTE FUNCTION messages.cleanup_typing_indicator();

-- Trigger to set expiration on disappearing messages
CREATE OR REPLACE FUNCTION messages.set_message_expiration()
RETURNS TRIGGER AS $$
DECLARE
    v_expire_after INTEGER;
BEGIN
    SELECT cs.disappearing_messages_duration INTO v_expire_after
    FROM messages.conversation_settings cs
    WHERE cs.conversation_id = NEW.conversation_id
    AND cs.disappearing_messages_enabled = TRUE;
    
    IF v_expire_after IS NOT NULL THEN
        NEW.expire_after_seconds = v_expire_after;
        NEW.expires_at = NOW() + (v_expire_after || ' seconds')::INTERVAL;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_messages_set_expiration
    BEFORE INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.set_message_expiration();

-- Trigger to update forward count
CREATE OR REPLACE FUNCTION messages.increment_forward_count()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.forwarded_from_message_id IS NOT NULL THEN
        UPDATE messages.messages
        SET forward_count = forward_count + 1
        WHERE id = NEW.forwarded_from_message_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_messages_increment_forward_count
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    WHEN (NEW.forwarded_from_message_id IS NOT NULL)
    EXECUTE FUNCTION messages.increment_forward_count();

-- Trigger to update participant accepted_at when status becomes accepted
CREATE OR REPLACE FUNCTION messages.set_participant_accepted_at()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status IS DISTINCT FROM OLD.status AND NEW.status = 'accepted' THEN
        -- This is handled differently, but keeping for reference
        NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to add creator as participant when conversation is created
CREATE OR REPLACE FUNCTION messages.add_creator_as_participant()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO messages.conversation_participants (
        conversation_id, user_id, role, joined_at
    ) VALUES (
        NEW.id, NEW.creator_user_id, 'owner', NOW()
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_conversations_add_creator
    AFTER INSERT ON messages.conversations
    FOR EACH ROW
    EXECUTE FUNCTION messages.add_creator_as_participant();