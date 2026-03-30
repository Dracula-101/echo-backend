-- =====================================================
-- MEDIA TRIGGERS — Processing Pipeline
-- =====================================================
--
-- Purpose:   Automatically queue media processing tasks
--            when new files are uploaded:
--              - Thumbnail generation for images and videos
--              - Virus scanning for ALL uploaded files
--
-- Depends:   functions/media.trigger_functions.sql
--              -> media.queue_thumbnail_generation()
--              -> media.queue_virus_scan()
--            functions/media.functions.sql
--              -> media.queue_file_processing()  (utility)
--
-- Tables:    media.files             (source — fires on INSERT)
--            media.processing_queue  (target — tasks enqueued)
--
-- Note:      Virus scanning runs at priority 10 (highest)
--            to ensure files are scanned before any other
--            processing occurs. Thumbnails run at priority 5.
--
-- =====================================================


-- Queue thumbnail generation at priority 5 for image and
-- video files. Requests small (150x150), medium (300x300),
-- and large (600x600) thumbnail sizes.
CREATE TRIGGER trigger_media_files_queue_thumbnail
    AFTER INSERT ON media.files
    FOR EACH ROW
    WHEN (NEW.file_category IN ('image', 'video'))
    EXECUTE FUNCTION media.queue_thumbnail_generation();

-- Queue virus scanning at priority 10 (highest) for ALL
-- uploaded files regardless of type. This ensures every
-- file is scanned for malware before it becomes accessible
-- to other users.
CREATE TRIGGER trigger_media_files_queue_scan
    AFTER INSERT ON media.files
    FOR EACH ROW
    EXECUTE FUNCTION media.queue_virus_scan();
