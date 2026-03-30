-- =====================================================
-- MESSAGES TRIGGERS — Full-Text Search Index
-- =====================================================
--
-- Purpose:   Maintain the messages.search_index table for
--            full-text search. Automatically creates or
--            updates tsvector entries when text messages
--            are inserted or edited.
--
-- Depends:   functions/messages.trigger_functions.sql
--              -> messages.update_search_index()
--
-- Tables:    messages.messages      (source — fires on INSERT/UPDATE)
--            messages.search_index  (target — tsvector upserted)
--
-- Note:      Only fires for text messages with non-null
--            content that have not been deleted. Edited
--            messages have their search vectors updated.
--
-- =====================================================


-- Insert or update the tsvector in messages.search_index
-- for full-text search. Uses English dictionary for stemming.
-- Fires on both INSERT (new messages) and UPDATE (edits).
CREATE TRIGGER trigger_messages_update_search_index
    AFTER INSERT OR UPDATE ON messages.messages
    FOR EACH ROW
    WHEN (NEW.message_type = 'text' AND NEW.content IS NOT NULL AND NOT NEW.is_deleted)
    EXECUTE FUNCTION messages.update_search_index();
