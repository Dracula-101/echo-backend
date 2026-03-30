-- =====================================================
-- MESSAGES TRIGGERS — Conversation Lifecycle
-- =====================================================
--
-- Purpose:   Manage the lifecycle of conversations:
--              - Create default conversation settings on creation
--              - Add the creator as the first participant
--              - Track member count as participants join/leave
--
-- Depends:   functions/messages.trigger_functions.sql
--              -> messages.create_default_conversation_settings()
--              -> messages.add_creator_as_participant()
--              -> messages.update_member_count()
--
-- Tables:    messages.conversations              (creation setup)
--            messages.conversation_participants   (member count)
--            messages.conversation_settings       (defaults)
--
-- =====================================================


-- -----------------------------------------------
-- CONVERSATION CREATION
-- -----------------------------------------------

-- Create a default conversation_settings row with all
-- settings at their DEFAULT values when a new conversation
-- is created.
CREATE TRIGGER trigger_conversations_create_settings
    AFTER INSERT ON messages.conversations
    FOR EACH ROW
    EXECUTE FUNCTION messages.create_default_conversation_settings();

-- Automatically add the conversation creator as the first
-- participant. Role is 'owner' for groups/channels, 'member'
-- for direct conversations.
CREATE TRIGGER trigger_conversations_add_creator
    AFTER INSERT ON messages.conversations
    FOR EACH ROW
    EXECUTE FUNCTION messages.add_creator_as_participant();


-- -----------------------------------------------
-- MEMBER COUNT TRACKING
-- -----------------------------------------------

-- Recalculate member_count when a new participant joins
CREATE TRIGGER trigger_participants_update_member_count_insert
    AFTER INSERT ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_member_count();

-- Recalculate member_count when a participant leaves or is removed
CREATE TRIGGER trigger_participants_update_member_count_update
    AFTER UPDATE ON messages.conversation_participants
    FOR EACH ROW
    WHEN (OLD.left_at IS DISTINCT FROM NEW.left_at OR OLD.removed_at IS DISTINCT FROM NEW.removed_at)
    EXECUTE FUNCTION messages.update_member_count();

-- Recalculate member_count when a participant record is deleted
CREATE TRIGGER trigger_participants_update_member_count_delete
    AFTER DELETE ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_member_count();
