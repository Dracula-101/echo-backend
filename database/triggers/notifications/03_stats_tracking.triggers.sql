-- =====================================================
-- NOTIFICATIONS TRIGGERS — Statistics Tracking
-- =====================================================
--
-- Purpose:   Maintain denormalized counters on
--            notifications.user_stats for efficient
--            analytics queries:
--              - Total notifications sent per user
--              - Push delivery and open counts
--              - Email delivery, open, and click counts
--
-- Depends:   functions/notifications.trigger_functions.sql
--              -> notifications.update_user_stats()
--              -> notifications.update_delivery_stats()
--              -> notifications.update_email_stats()
--
-- Tables:    notifications.notifications        (source for send stats)
--            notifications.push_delivery_log    (source for push stats)
--            notifications.email_notifications  (source for email stats)
--            notifications.user_stats           (target — counters updated)
--
-- =====================================================


-- -----------------------------------------------
-- USER NOTIFICATION STATS
-- -----------------------------------------------

-- Increment total_notifications_sent (and push_sent for
-- mobile platforms) whenever a new notification is created
CREATE TRIGGER trigger_notifications_update_stats_insert
    AFTER INSERT ON notifications.notifications
    FOR EACH ROW
    EXECUTE FUNCTION notifications.update_user_stats();


-- -----------------------------------------------
-- PUSH DELIVERY STATS
-- -----------------------------------------------

-- Increment push delivery/opened counters when a push
-- delivery log entry transitions to delivered or opened
CREATE TRIGGER trigger_push_delivery_update_stats
    AFTER UPDATE ON notifications.push_delivery_log
    FOR EACH ROW
    WHEN (OLD.status IS DISTINCT FROM NEW.status OR OLD.opened_at IS DISTINCT FROM NEW.opened_at)
    EXECUTE FUNCTION notifications.update_delivery_stats();


-- -----------------------------------------------
-- EMAIL DELIVERY STATS
-- -----------------------------------------------

-- Increment email delivery/opened/clicked counters when
-- an email notification's status, opened_at, or clicked_at
-- changes
CREATE TRIGGER trigger_email_notifications_update_stats
    AFTER UPDATE ON notifications.email_notifications
    FOR EACH ROW
    WHEN (
        OLD.status IS DISTINCT FROM NEW.status
        OR OLD.opened_at IS DISTINCT FROM NEW.opened_at
        OR OLD.clicked_at IS DISTINCT FROM NEW.clicked_at
    )
    EXECUTE FUNCTION notifications.update_email_stats();
