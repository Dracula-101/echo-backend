-- =====================================================
-- NOTIFICATIONS TRIGGERS — Announcements
-- =====================================================
--
-- Purpose:   Maintain announcement engagement metrics
--            and enforce business rules:
--              - Increment view_count on first/subsequent views
--              - Increment click_count when a user clicks
--              - Increment dismiss_count when a user dismisses
--              - Validate conversation channel overrides
--
-- Depends:   functions/notifications.trigger_functions.sql
--              -> notifications.increment_announcement_views()
--              -> notifications.increment_announcement_clicks()
--              -> notifications.validate_conversation_channel()
--
-- Tables:    notifications.announcement_views     (source for view/click/dismiss)
--            notifications.announcements           (target — counters updated)
--            notifications.conversation_channels   (validated on insert/update)
--            messages.conversation_participants    (checked for membership)
--
-- =====================================================


-- -----------------------------------------------
-- VIEW TRACKING
-- -----------------------------------------------

-- Increment the announcement's view_count every time
-- a new announcement_views row is created (first view
-- by a user)
CREATE TRIGGER trigger_announcement_views_increment
    AFTER INSERT ON notifications.announcement_views
    FOR EACH ROW
    EXECUTE FUNCTION notifications.increment_announcement_views();


-- -----------------------------------------------
-- CLICK & DISMISS TRACKING
-- -----------------------------------------------

-- Increment click_count and/or dismiss_count on the
-- announcement when a user's clicked or dismissed
-- flag transitions from FALSE to TRUE
CREATE TRIGGER trigger_announcement_views_clicks
    AFTER UPDATE ON notifications.announcement_views
    FOR EACH ROW
    WHEN (OLD.clicked IS DISTINCT FROM NEW.clicked OR OLD.dismissed IS DISTINCT FROM NEW.dismissed)
    EXECUTE FUNCTION notifications.increment_announcement_clicks();


-- -----------------------------------------------
-- CONVERSATION CHANNEL VALIDATION
-- -----------------------------------------------

-- Before creating or updating per-conversation notification
-- overrides, verify that the user is an active participant
-- in the conversation (not left or removed). Raises an
-- exception if the user is not a valid participant.
CREATE TRIGGER trigger_conversation_channels_validate
    BEFORE INSERT OR UPDATE ON notifications.conversation_channels
    FOR EACH ROW
    EXECUTE FUNCTION notifications.validate_conversation_channel();
