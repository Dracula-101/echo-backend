-- =====================================================
-- MESSAGES TRIGGERS — Timestamp Management
-- =====================================================
--
-- Purpose:   Automatically maintain `updated_at` columns
--            on messages schema tables whenever rows are
--            modified.
--
-- Depends:   functions/messages.trigger_functions.sql
--              -> messages.update_updated_at_column()
--
-- Tables:    messages.conversations
--            messages.conversation_participants
--            messages.messages
--            messages.message_reports
--            messages.drafts
--
-- =====================================================


-- Keep updated_at current on conversations
CREATE TRIGGER trigger_messages_conversations_updated_at
    BEFORE UPDATE ON messages.conversations
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();

-- Keep updated_at current on conversation participants
CREATE TRIGGER trigger_messages_participants_updated_at
    BEFORE UPDATE ON messages.conversation_participants
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();

-- Keep updated_at current on messages (e.g. edits, pin status)
CREATE TRIGGER trigger_messages_updated_at
    BEFORE UPDATE ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();

-- Keep updated_at current on message reports
CREATE TRIGGER trigger_messages_reports_updated_at
    BEFORE UPDATE ON messages.message_reports
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();

-- Keep updated_at current on message drafts
CREATE TRIGGER trigger_messages_drafts_updated_at
    BEFORE UPDATE ON messages.drafts
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_updated_at_column();
