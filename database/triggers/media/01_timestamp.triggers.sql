-- =====================================================
-- MEDIA TRIGGERS — Timestamp Management
-- =====================================================
--
-- Purpose:   Automatically maintain `updated_at` columns
--            on media schema tables whenever rows are modified.
--
-- Depends:   functions/media.trigger_functions.sql
--              -> media.update_updated_at_column()
--
-- Tables:    media.files
--            media.albums
--            media.sticker_packs
--            media.processing_queue
--            media.transcoding_jobs
--            media.storage_stats
--
-- =====================================================


-- Keep updated_at current on media files
CREATE TRIGGER trigger_media_files_updated_at
    BEFORE UPDATE ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();

-- Keep updated_at current on media albums
CREATE TRIGGER trigger_media_albums_updated_at
    BEFORE UPDATE ON media.albums
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();

-- Keep updated_at current on sticker packs
CREATE TRIGGER trigger_media_sticker_packs_updated_at
    BEFORE UPDATE ON media.sticker_packs
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();

-- Keep updated_at current on processing queue items
CREATE TRIGGER trigger_media_processing_queue_updated_at
    BEFORE UPDATE ON media.processing_queue
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();

-- Keep updated_at current on transcoding jobs
CREATE TRIGGER trigger_media_transcoding_jobs_updated_at
    BEFORE UPDATE ON media.transcoding_jobs
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();

-- Keep updated_at current on storage statistics
CREATE TRIGGER trigger_media_storage_stats_updated_at
    BEFORE UPDATE ON media.storage_stats
    FOR EACH ROW
    EXECUTE FUNCTION media.update_updated_at_column();
