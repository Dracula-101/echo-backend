-- =====================================================
-- USERS TRIGGERS — Status & Activity
-- =====================================================
--
-- Purpose:   Track engagement with user status updates
--            (stories). Increments view counters when
--            users view each other's statuses.
--
-- Depends:   functions/users.trigger_functions.sql
--              -> users.increment_status_views()
--
-- Tables:    users.status_views   (source — fires on INSERT)
--            users.status_history (target — views_count updated)
--
-- =====================================================


-- When a user views someone's status, increment the
-- views_count counter on the corresponding status_history row.
CREATE TRIGGER trigger_users_status_views_increment
    AFTER INSERT ON users.status_views
    FOR EACH ROW
    EXECUTE FUNCTION users.increment_status_views();
