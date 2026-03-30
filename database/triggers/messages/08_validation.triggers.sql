-- =====================================================
-- MESSAGES TRIGGERS — Message Validation & Cleanup
-- =====================================================
--
-- Purpose:   Enforce messaging business rules and clean up
--            ephemeral data:
--              - Validate sender is an active participant
--                with send permissions
--              - Prevent blocked users from messaging each
--                other in direct conversations
--              - Clean up expired typing indicators
--
-- Depends:   functions/messages.trigger_functions.sql
--              -> messages.validate_participant_can_send()
--              -> messages.validate_not_blocked()
--              -> messages.cleanup_typing_indicator()
--            functions/users.functions.sql
--              -> users.is_blocked()  (utility)
--
-- Tables:    messages.messages            (send validation)
--            messages.typing_indicators   (cleanup)
--            messages.conversation_participants (checked)
--            users.blocked_users          (checked)
--
-- Note:      These are BEFORE triggers that reject invalid
--            operations by raising exceptions.
--
-- =====================================================


-- -----------------------------------------------
-- SEND PERMISSION VALIDATION
-- -----------------------------------------------

-- Ensure the sender is an active participant in the
-- conversation with can_send_messages = TRUE. Raises an
-- exception if the user has left, been removed, or had
-- their send permission revoked.
CREATE TRIGGER trigger_messages_validate_sender
    BEFORE INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.validate_participant_can_send();

-- For direct (1:1) conversations, check whether either
-- party has blocked the other. Raises an exception if a
-- block exists in either direction.
CREATE TRIGGER trigger_messages_validate_not_blocked
    BEFORE INSERT ON messages.messages
    FOR EACH ROW
    EXECUTE FUNCTION messages.validate_not_blocked();


-- -----------------------------------------------
-- TYPING INDICATOR CLEANUP
-- -----------------------------------------------

-- Before inserting a new typing indicator, remove all
-- expired indicators across all conversations. This keeps
-- the typing_indicators table lean without requiring a
-- separate cleanup job.
CREATE TRIGGER trigger_typing_indicators_cleanup
    BEFORE INSERT ON messages.typing_indicators
    FOR EACH ROW
    EXECUTE FUNCTION messages.cleanup_typing_indicator();
