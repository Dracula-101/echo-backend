-- =====================================================
-- NOTIFICATIONS TRIGGERS — Delivery Lifecycle
-- =====================================================
--
-- Purpose:   Track notification delivery through its
--            lifecycle stages:
--              - Sync push delivery status back to the
--                parent notification record
--              - Update batch progress counters as
--                individual notifications are delivered,
--                read, or fail
--
-- Depends:   functions/notifications.trigger_functions.sql
--              -> notifications.update_notification_delivery_status()
--              -> notifications.update_batch_progress()
--
-- Tables:    notifications.push_delivery_log (source — status changes)
--            notifications.notifications     (target — delivery status synced;
--                                             source for batch progress)
--            notifications.batches           (target — progress counters)
--
-- =====================================================


-- -----------------------------------------------
-- PUSH → NOTIFICATION STATUS SYNC
-- -----------------------------------------------

-- When a push delivery log entry transitions to
-- 'delivered' or 'failed', propagate that status back
-- to the parent notification record along with the
-- delivered_at timestamp or failure reason
CREATE TRIGGER trigger_push_delivery_update_notification
    AFTER UPDATE ON notifications.push_delivery_log
    FOR EACH ROW
    WHEN (OLD.status IS DISTINCT FROM NEW.status)
    EXECUTE FUNCTION notifications.update_notification_delivery_status();


-- -----------------------------------------------
-- BATCH PROGRESS TRACKING
-- -----------------------------------------------

-- When a notification's delivery_status or is_read
-- changes, update the sent/delivered/failed/opened
-- counters on the parent batch record (if the
-- notification belongs to a batch via metadata)
CREATE TRIGGER trigger_notifications_batch_progress
    AFTER UPDATE ON notifications.notifications
    FOR EACH ROW
    WHEN (
        OLD.delivery_status IS DISTINCT FROM NEW.delivery_status
        OR OLD.is_read IS DISTINCT FROM NEW.is_read
    )
    EXECUTE FUNCTION notifications.update_batch_progress();
