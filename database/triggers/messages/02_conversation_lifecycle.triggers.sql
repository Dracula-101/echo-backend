-- =====================================================
-- MESSAGES TRIGGERS — 02: Conversation Lifecycle
-- =====================================================


-- -----------------------------------------------
-- CONVERSATIONS — updated_at
-- -----------------------------------------------

CREATE TRIGGER trigger_conversations_updated_at
    BEFORE UPDATE ON messages.conversations
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();


-- -----------------------------------------------
-- CONVERSATIONS — creation setup
-- -----------------------------------------------

-- Create a default conversation_settings row with
-- all settings at their DEFAULT values when a new
-- conversation is inserted.
CREATE TRIGGER trigger_conversations_create_settings
    AFTER INSERT ON messages.conversations
    FOR EACH ROW
    EXECUTE FUNCTION messages.create_default_conversation_settings();

-- Automatically add the conversation creator as the
-- first participant.
--   'owner'  — group / channel / broadcast
--   'member' — direct conversation
-- This fires AFTER trigger_conversations_create_settings
-- so the settings row exists before the participant
-- insert cascades.
CREATE TRIGGER trigger_conversations_add_creator
    AFTER INSERT ON messages.conversations
    FOR EACH ROW
    EXECUTE FUNCTION messages.add_creator_as_participant();


-- -----------------------------------------------
-- PARTICIPANTS — updated_at
-- -----------------------------------------------

CREATE TRIGGER trigger_participants_updated_at
    BEFORE UPDATE ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();


-- -----------------------------------------------
-- PARTICIPANTS — permission management
-- -----------------------------------------------

-- Set all permission columns (can_send_messages,
-- can_add_members, etc.) based on the participant's
-- role. Fires before INSERT and before any UPDATE
-- that changes the role column, so permissions are
-- always in sync with the role.
CREATE TRIGGER trigger_participants_set_permissions
    BEFORE INSERT OR UPDATE OF role ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.set_participant_permissions();


-- -----------------------------------------------
-- PARTICIPANTS — member count tracking
-- -----------------------------------------------

-- Recalculate member_count when a new participant
-- joins (any INSERT).
CREATE TRIGGER trigger_participants_member_count_insert
    AFTER INSERT ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_member_count();

-- Recalculate member_count when a participant's
-- left_at or removed_at changes (soft leave/removal).
CREATE TRIGGER trigger_participants_member_count_update
    AFTER UPDATE ON messages.conversation_participants
    FOR EACH ROW
    WHEN (
        OLD.left_at     IS DISTINCT FROM NEW.left_at
        OR
        OLD.removed_at  IS DISTINCT FROM NEW.removed_at
    )
    EXECUTE FUNCTION messages.update_member_count();

-- Recalculate member_count when a participant record
-- is hard-deleted.
CREATE TRIGGER trigger_participants_member_count_delete
    AFTER DELETE ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_member_count();