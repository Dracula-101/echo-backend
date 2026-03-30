-- =====================================================
-- MEDIA TRIGGERS — Storage Management
-- =====================================================
--
-- Purpose:   Manage user storage lifecycle:
--              - Validate upload against storage quota
--              - Recalculate storage stats on file changes
--              - Create default storage stats for new users
--              - Schedule permanent deletion after soft delete
--
-- Depends:   functions/media.trigger_functions.sql
--              -> media.validate_storage_quota()
--              -> media.trigger_update_storage_stats()
--              -> media.create_default_storage_stats()
--              -> media.set_permanent_deletion_date()
--            functions/media.functions.sql
--              -> media.check_storage_quota()  (utility)
--              -> media.update_storage_stats() (utility)
--
-- Tables:    media.files          (quota check, stats, soft delete)
--            media.storage_stats  (recalculated)
--            auth.users           (new user setup)
--
-- =====================================================


-- -----------------------------------------------
-- QUOTA ENFORCEMENT
-- -----------------------------------------------

-- Before a file is uploaded, verify that the user's
-- remaining storage quota can accommodate the file size.
-- Raises an exception if the quota would be exceeded
-- (default quota: 5 GB per user).
CREATE TRIGGER trigger_media_files_validate_quota
    BEFORE INSERT ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.validate_storage_quota();


-- -----------------------------------------------
-- STORAGE STATISTICS
-- -----------------------------------------------

-- Recalculate the user's storage stats when a new file is uploaded
CREATE TRIGGER trigger_media_files_storage_stats_insert
    AFTER INSERT ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.trigger_update_storage_stats();

-- Recalculate when a file is soft-deleted or its size changes
CREATE TRIGGER trigger_media_files_storage_stats_update
    AFTER UPDATE ON media.files
    FOR EACH ROW
    WHEN (OLD.deleted_at IS DISTINCT FROM NEW.deleted_at OR OLD.file_size_bytes IS DISTINCT FROM NEW.file_size_bytes)
    EXECUTE FUNCTION media.trigger_update_storage_stats();

-- Recalculate when a file is permanently deleted
CREATE TRIGGER trigger_media_files_storage_stats_delete
    AFTER DELETE ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.trigger_update_storage_stats();


-- -----------------------------------------------
-- NEW USER SETUP
-- -----------------------------------------------

-- Create a default storage_stats row with the 5 GB quota
-- and zero usage for every new user.
CREATE TRIGGER trigger_auth_users_create_storage_stats
    AFTER INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION media.create_default_storage_stats();


-- -----------------------------------------------
-- SOFT DELETE LIFECYCLE
-- -----------------------------------------------

-- When a file is soft-deleted (deleted_at set), schedule
-- permanent deletion 30 days later by setting
-- permanently_delete_at. A separate cleanup job
-- (media.cleanup_deleted_files) handles the actual purge.
CREATE TRIGGER trigger_media_files_set_deletion_date
    BEFORE UPDATE ON media.files
    FOR EACH ROW
    WHEN (NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL)
    EXECUTE FUNCTION media.set_permanent_deletion_date();
