-- =====================================================
-- MEDIA TRIGGERS — Count Tracking
-- =====================================================
--
-- Purpose:   Maintain denormalized counters across media
--            entities for efficient querying:
--              - File access and download/view counts
--              - Album file counts
--              - Sticker pack sticker counts
--              - Tag usage counts
--              - Share view/download counts
--
-- Depends:   functions/media.trigger_functions.sql
--              -> media.increment_access_count()
--              -> media.increment_download_count()
--              -> media.update_album_file_count()
--              -> media.update_sticker_pack_count()
--              -> media.update_tag_usage_count()
--              -> media.increment_share_counts()
--
-- Tables:    media.access_log     (source for access/download counts)
--            media.album_files    (source for album counts)
--            media.stickers       (source for sticker pack counts)
--            media.file_tags      (source for tag usage counts)
--            media.files          (target — counters updated)
--            media.albums         (target — file_count updated)
--            media.sticker_packs  (target — sticker_count updated)
--            media.tags           (target — usage_count updated)
--            media.shares         (target — view/download counts)
--
-- =====================================================


-- -----------------------------------------------
-- FILE ACCESS TRACKING
-- -----------------------------------------------

-- Increment the access_count and update last_accessed_at
-- on the file for every access log entry
CREATE TRIGGER trigger_media_access_log_increment
    AFTER INSERT ON media.access_log
    FOR EACH ROW
    EXECUTE FUNCTION media.increment_access_count();

-- Increment the download_count or view_count on the file
-- depending on the access type
CREATE TRIGGER trigger_media_access_log_download
    AFTER INSERT ON media.access_log
    FOR EACH ROW
    WHEN (NEW.access_type IN ('download', 'view'))
    EXECUTE FUNCTION media.increment_download_count();

-- Increment view or download counts on active share records
-- when the shared file is accessed
CREATE TRIGGER trigger_media_access_log_share_counts
    AFTER INSERT ON media.access_log
    FOR EACH ROW
    WHEN (NEW.access_type IN ('view', 'download'))
    EXECUTE FUNCTION media.increment_share_counts();


-- -----------------------------------------------
-- ALBUM FILE COUNTS
-- -----------------------------------------------

-- Recalculate album file_count when a file is added to an album
CREATE TRIGGER trigger_media_album_files_count_insert
    AFTER INSERT ON media.album_files
    FOR EACH ROW
    EXECUTE FUNCTION media.update_album_file_count();

-- Recalculate album file_count when a file is removed from an album
CREATE TRIGGER trigger_media_album_files_count_delete
    AFTER DELETE ON media.album_files
    FOR EACH ROW
    EXECUTE FUNCTION media.update_album_file_count();


-- -----------------------------------------------
-- STICKER PACK COUNTS
-- -----------------------------------------------

-- Recalculate sticker_count when a new sticker is added to a pack
CREATE TRIGGER trigger_media_stickers_count_insert
    AFTER INSERT ON media.stickers
    FOR EACH ROW
    EXECUTE FUNCTION media.update_sticker_pack_count();

-- Recalculate sticker_count when a sticker's is_active status changes
CREATE TRIGGER trigger_media_stickers_count_update
    AFTER UPDATE ON media.stickers
    FOR EACH ROW
    WHEN (OLD.is_active IS DISTINCT FROM NEW.is_active)
    EXECUTE FUNCTION media.update_sticker_pack_count();

-- Recalculate sticker_count when a sticker is deleted from a pack
CREATE TRIGGER trigger_media_stickers_count_delete
    AFTER DELETE ON media.stickers
    FOR EACH ROW
    EXECUTE FUNCTION media.update_sticker_pack_count();


-- -----------------------------------------------
-- TAG USAGE COUNTS
-- -----------------------------------------------

-- Recalculate tag usage_count when a tag is applied to a file
CREATE TRIGGER trigger_media_file_tags_usage_insert
    AFTER INSERT ON media.file_tags
    FOR EACH ROW
    EXECUTE FUNCTION media.update_tag_usage_count();

-- Recalculate tag usage_count when a tag is removed from a file
CREATE TRIGGER trigger_media_file_tags_usage_delete
    AFTER DELETE ON media.file_tags
    FOR EACH ROW
    EXECUTE FUNCTION media.update_tag_usage_count();
