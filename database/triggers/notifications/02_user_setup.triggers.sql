-- =====================================================
-- NOTIFICATIONS TRIGGERS — User Registration Setup
-- =====================================================
--
-- Purpose:   Automatically provision notification-related
--            rows when a new user registers:
--              - Default notification preferences (all
--                channels enabled at sensible defaults)
--              - Default notification statistics row
--                (all counters initialized to zero)
--
-- Depends:   functions/notifications.trigger_functions.sql
--              -> notifications.create_default_preferences()
--              -> notifications.create_default_notification_stats()
--
-- Tables:    auth.users                    (source — fires on INSERT)
--            notifications.user_preferences (target — row created)
--            notifications.user_stats       (target — row created)
--
-- Note:      Both triggers use ON CONFLICT DO NOTHING to
--            safely handle duplicate registration events
--            (e.g. retry after partial failure).
--
-- =====================================================


-- Create default notification preferences for every
-- new user with all preference columns at their table
-- DEFAULT values (push enabled, email enabled, etc.)
CREATE TRIGGER trigger_auth_users_create_notification_preferences
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION notifications.create_default_preferences();

-- Create default notification statistics row for every
-- new user with all counters initialized to zero
CREATE TRIGGER trigger_auth_users_create_notification_stats
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION notifications.create_default_notification_stats();
