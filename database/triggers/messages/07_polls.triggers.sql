-- =====================================================
-- MESSAGES TRIGGERS — Poll Management
-- =====================================================
--
-- Purpose:   Manage the lifecycle and integrity of polls:
--              - Update vote counts and percentages on vote
--              - Auto-close polls past their closing time
--              - Prevent voting on closed polls
--              - Enforce single-vote rule on non-multi polls
--
-- Depends:   functions/messages.trigger_functions.sql
--              -> messages.update_poll_votes()
--              -> messages.auto_close_poll()
--              -> messages.validate_poll_not_closed()
--              -> messages.validate_single_vote()
--
-- Tables:    messages.poll_votes    (vote triggers)
--            messages.polls         (auto-close)
--            messages.poll_options  (counts updated)
--
-- =====================================================


-- -----------------------------------------------
-- VOTE COUNT TRACKING
-- -----------------------------------------------

-- Recalculate vote counts and percentages when a vote is cast
CREATE TRIGGER trigger_poll_votes_update_counts
    AFTER INSERT ON messages.poll_votes
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_poll_votes();

-- Recalculate vote counts and percentages when a vote is removed
CREATE TRIGGER trigger_poll_votes_update_counts_delete
    AFTER DELETE ON messages.poll_votes
    FOR EACH ROW
    EXECUTE FUNCTION messages.update_poll_votes();


-- -----------------------------------------------
-- POLL LIFECYCLE
-- -----------------------------------------------

-- Automatically close a poll when it is accessed after
-- its closes_at timestamp has passed.
CREATE TRIGGER trigger_polls_auto_close
    BEFORE UPDATE ON messages.polls
    FOR EACH ROW
    WHEN (NEW.closes_at IS NOT NULL AND NEW.closes_at <= NOW())
    EXECUTE FUNCTION messages.auto_close_poll();


-- -----------------------------------------------
-- VOTE VALIDATION
-- -----------------------------------------------

-- Prevent casting votes on polls that have already been closed
CREATE TRIGGER trigger_poll_votes_validate_not_closed
    BEFORE INSERT ON messages.poll_votes
    FOR EACH ROW
    EXECUTE FUNCTION messages.validate_poll_not_closed();

-- For polls that do not allow multiple answers, prevent
-- a user from casting more than one vote
CREATE TRIGGER trigger_poll_votes_validate_single
    BEFORE INSERT ON messages.poll_votes
    FOR EACH ROW
    EXECUTE FUNCTION messages.validate_single_vote();
