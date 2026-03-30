-- =====================================================
-- MESSAGES TRIGGERS — Message Lifecycle
-- =====================================================
--
-- Purpose:   Manage message state transitions and metadata:
--              - Update conversation stats on new message
--              - Track message edits with history
--              - Set expiration for disappearing messages
--              - Track reply counts for threaded conversations
--              - Track forward counts for message virality
--
-- Depends:   functions/messages.trigger_functions.sql
--              -> messages.update_conversation_last_message()
--              -> messages.set_edited_timestamp()
--              -> messages.set_message_expiration()
--              -> messages.update_reply_count()
--              -> messages.increment_forward_count()
--
-- Tables:    messages.messages        (all triggers)
--            messages.conversations   (stats updated)
--            messages.conversation_settings (expiration config)
--
-- =====================================================


-- -----------------------------------------------
-- NEW MESSAGE HANDLING
-- -----------------------------------------------

-- Update the parent conversation's last_message_id,
-- last_message_at, last_activity_at, and increment
-- message_count when a new message is sent.
CREATE TRIGGER trigger_messages_update_conversation
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_conversation_last_message();

-- If the conversation has disappearing messages enabled,
-- calculate and set the expires_at timestamp on new messages
-- based on the configured duration.
CREATE TRIGGER trigger_messages_set_expiration
    BEFORE INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.set_message_expiration();


-- -----------------------------------------------
-- MESSAGE EDITING
-- -----------------------------------------------

-- When a message's content is modified, mark it as edited,
-- set the edited_at timestamp, and append the previous
-- content to the edit_history JSONB array for audit trail.
CREATE TRIGGER trigger_messages_set_edited
    BEFORE UPDATE ON messages.messages
    FOR EACH ROW
    WHEN (OLD.content IS DISTINCT FROM NEW.content)
    EXECUTE FUNCTION messages.set_edited_timestamp();


-- -----------------------------------------------
-- THREADING & FORWARDING
-- -----------------------------------------------

-- When a reply is sent (parent_message_id is set),
-- increment the reply_count and update last_reply_at
-- on the parent message.
CREATE TRIGGER trigger_messages_update_reply_count
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    WHEN (NEW.parent_message_id IS NOT NULL)
    EXECUTE FUNCTION messages.update_reply_count();

-- When a message is forwarded (forwarded_from_message_id is set),
-- increment the forward_count on the original message to
-- track how widely it has been shared.
CREATE TRIGGER trigger_messages_increment_forward_count
    AFTER INSERT ON messages.messages
    FOR EACH ROW
    WHEN (NEW.forwarded_from_message_id IS NOT NULL)
    EXECUTE FUNCTION messages.increment_forward_count();
