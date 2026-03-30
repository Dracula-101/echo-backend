-- =====================================================
-- MESSAGES TRIGGERS — Reaction Tracking
-- =====================================================
--
-- Purpose:   Track reaction counts on messages. Recalculates
--            the reaction_count on the parent message when
--            reactions are added or removed.
--
-- Depends:   functions/messages.trigger_functions.sql
--              -> messages.update_reaction_count()
--
-- Tables:    messages.reactions  (source — fires on INSERT/DELETE)
--            messages.messages   (target — reaction_count updated)
--
-- =====================================================


-- Recalculate reaction_count when a reaction is added
CREATE TRIGGER trigger_reactions_update_count_insert
    AFTER INSERT ON messages.reactions
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_reaction_count();

-- Recalculate reaction_count when a reaction is removed
CREATE TRIGGER trigger_reactions_update_count_delete
    AFTER DELETE ON messages.reactions
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_reaction_count();
