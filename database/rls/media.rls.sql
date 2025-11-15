-- =====================================================
-- MEDIA SCHEMA - ROW LEVEL SECURITY (RLS)
-- =====================================================

-- Enable RLS on all media tables
ALTER TABLE media.files ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.processing_queue ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.thumbnails ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.transcoding_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.albums ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.album_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.file_tags ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.shares ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.access_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.sticker_packs ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.stickers ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.user_sticker_packs ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.gifs ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.favorite_gifs ENABLE ROW LEVEL SECURITY;
ALTER TABLE media.storage_stats ENABLE ROW LEVEL SECURITY;

-- =====================================================
-- FILES TABLE POLICIES
-- =====================================================

-- Users can view their own files
CREATE POLICY files_select_own
    ON media.files
    FOR SELECT
    USING (uploader_user_id = auth.current_user_id() AND deleted_at IS NULL);

-- Users can view public files
CREATE POLICY files_select_public
    ON media.files
    FOR SELECT
    USING (visibility = 'public' AND deleted_at IS NULL);

-- Users can view files shared with them
CREATE POLICY files_select_shared
    ON media.files
    FOR SELECT
    USING (
        deleted_at IS NULL
        AND EXISTS (
            SELECT 1 FROM media.shares s
            WHERE s.file_id = media.files.id
            AND (
                s.shared_with_user_id = auth.current_user_id()
                OR EXISTS (
                    SELECT 1 FROM messages.conversation_participants cp
                    WHERE cp.conversation_id = s.shared_with_conversation_id
                    AND cp.user_id = auth.current_user_id()
                    AND cp.left_at IS NULL
                )
            )
            AND s.is_active = TRUE
            AND (s.expires_at IS NULL OR s.expires_at > NOW())
        )
    );

-- Users can insert their own files
CREATE POLICY files_insert_own
    ON media.files
    FOR INSERT
    WITH CHECK (uploader_user_id = auth.current_user_id());

-- Users can update their own files
CREATE POLICY files_update_own
    ON media.files
    FOR UPDATE
    USING (uploader_user_id = auth.current_user_id())
    WITH CHECK (uploader_user_id = auth.current_user_id());

-- Users can delete their own files
CREATE POLICY files_delete_own
    ON media.files
    FOR DELETE
    USING (uploader_user_id = auth.current_user_id());

-- Admins can manage all files
CREATE POLICY files_admin_all
    ON media.files
    FOR ALL
    USING (auth.is_admin());

-- =====================================================
-- PROCESSING QUEUE POLICIES
-- =====================================================

-- Service role can manage processing queue
CREATE POLICY processing_queue_service_all
    ON media.processing_queue
    FOR ALL
    USING (TRUE);

-- =====================================================
-- THUMBNAILS POLICIES
-- =====================================================

-- Users can view thumbnails of files they can access
CREATE POLICY thumbnails_select
    ON media.thumbnails
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM media.files f
            WHERE f.id = media.thumbnails.file_id
            AND (
                f.uploader_user_id = auth.current_user_id()
                OR f.visibility = 'public'
            )
        )
    );

-- Service role can manage thumbnails
CREATE POLICY thumbnails_service_all
    ON media.thumbnails
    FOR ALL
    USING (TRUE);

-- =====================================================
-- TRANSCODING JOBS POLICIES
-- =====================================================

-- Service role can manage transcoding jobs
CREATE POLICY transcoding_jobs_service_all
    ON media.transcoding_jobs
    FOR ALL
    USING (TRUE);

-- =====================================================
-- ALBUMS POLICIES
-- =====================================================

-- Users can view their own albums
CREATE POLICY albums_select_own
    ON media.albums
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Users can view public albums
CREATE POLICY albums_select_public
    ON media.albums
    FOR SELECT
    USING (visibility = 'public');

-- Users can manage their own albums
CREATE POLICY albums_insert_own
    ON media.albums
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id());

CREATE POLICY albums_update_own
    ON media.albums
    FOR UPDATE
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

CREATE POLICY albums_delete_own
    ON media.albums
    FOR DELETE
    USING (user_id = auth.current_user_id());

-- =====================================================
-- ALBUM FILES POLICIES
-- =====================================================

-- Users can manage files in their own albums
CREATE POLICY album_files_all_own
    ON media.album_files
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM media.albums a
            WHERE a.id = media.album_files.album_id
            AND a.user_id = auth.current_user_id()
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM media.albums a
            WHERE a.id = media.album_files.album_id
            AND a.user_id = auth.current_user_id()
        )
    );

-- =====================================================
-- TAGS POLICIES
-- =====================================================

-- Users can view all tags
CREATE POLICY tags_select_all
    ON media.tags
    FOR SELECT
    USING (TRUE);

-- Users can create their own tags
CREATE POLICY tags_insert_own
    ON media.tags
    FOR INSERT
    WITH CHECK (user_id = auth.current_user_id() OR user_id IS NULL);

-- =====================================================
-- FILE TAGS POLICIES
-- =====================================================

-- Users can view tags on files they can access
CREATE POLICY file_tags_select
    ON media.file_tags
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM media.files f
            WHERE f.id = media.file_tags.file_id
            AND (
                f.uploader_user_id = auth.current_user_id()
                OR f.visibility = 'public'
            )
        )
    );

-- Users can tag their own files
CREATE POLICY file_tags_insert_own
    ON media.file_tags
    FOR INSERT
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM media.files f
            WHERE f.id = media.file_tags.file_id
            AND f.uploader_user_id = auth.current_user_id()
        )
    );

CREATE POLICY file_tags_delete_own
    ON media.file_tags
    FOR DELETE
    USING (
        EXISTS (
            SELECT 1 FROM media.files f
            WHERE f.id = media.file_tags.file_id
            AND f.uploader_user_id = auth.current_user_id()
        )
    );

-- =====================================================
-- SHARES POLICIES
-- =====================================================

-- Users can view shares they created
CREATE POLICY shares_select_own
    ON media.shares
    FOR SELECT
    USING (shared_by_user_id = auth.current_user_id());

-- Users can view shares shared with them
CREATE POLICY shares_select_shared_with
    ON media.shares
    FOR SELECT
    USING (
        shared_with_user_id = auth.current_user_id()
        OR EXISTS (
            SELECT 1 FROM messages.conversation_participants cp
            WHERE cp.conversation_id = media.shares.shared_with_conversation_id
            AND cp.user_id = auth.current_user_id()
        )
    );

-- Users can create shares for their files
CREATE POLICY shares_insert_own
    ON media.shares
    FOR INSERT
    WITH CHECK (
        shared_by_user_id = auth.current_user_id()
        AND EXISTS (
            SELECT 1 FROM media.files f
            WHERE f.id = media.shares.file_id
            AND f.uploader_user_id = auth.current_user_id()
        )
    );

-- Users can update/delete their shares
CREATE POLICY shares_update_own
    ON media.shares
    FOR UPDATE
    USING (shared_by_user_id = auth.current_user_id())
    WITH CHECK (shared_by_user_id = auth.current_user_id());

CREATE POLICY shares_delete_own
    ON media.shares
    FOR DELETE
    USING (shared_by_user_id = auth.current_user_id());

-- =====================================================
-- ACCESS LOG POLICIES
-- =====================================================

-- Users can view access logs for their files
CREATE POLICY access_log_select_own
    ON media.access_log
    FOR SELECT
    USING (
        user_id = auth.current_user_id()
        OR EXISTS (
            SELECT 1 FROM media.files f
            WHERE f.id = media.access_log.file_id
            AND f.uploader_user_id = auth.current_user_id()
        )
    );

-- Service role can insert access logs
CREATE POLICY access_log_insert_service
    ON media.access_log
    FOR INSERT
    WITH CHECK (TRUE);

-- =====================================================
-- STICKER PACKS POLICIES
-- =====================================================

-- Users can view public sticker packs
CREATE POLICY sticker_packs_select_public
    ON media.sticker_packs
    FOR SELECT
    USING (is_public = TRUE);

-- Users can view their own sticker packs
CREATE POLICY sticker_packs_select_own
    ON media.sticker_packs
    FOR SELECT
    USING (creator_user_id = auth.current_user_id());

-- Users can create sticker packs
CREATE POLICY sticker_packs_insert_own
    ON media.sticker_packs
    FOR INSERT
    WITH CHECK (creator_user_id = auth.current_user_id());

-- Users can update their own sticker packs
CREATE POLICY sticker_packs_update_own
    ON media.sticker_packs
    FOR UPDATE
    USING (creator_user_id = auth.current_user_id())
    WITH CHECK (creator_user_id = auth.current_user_id());

-- =====================================================
-- STICKERS POLICIES
-- =====================================================

-- Users can view stickers from accessible packs
CREATE POLICY stickers_select
    ON media.stickers
    FOR SELECT
    USING (
        is_active = TRUE
        AND EXISTS (
            SELECT 1 FROM media.sticker_packs sp
            WHERE sp.id = media.stickers.sticker_pack_id
            AND (sp.is_public = TRUE OR sp.creator_user_id = auth.current_user_id())
        )
    );

-- Users can manage stickers in their packs
CREATE POLICY stickers_manage_own
    ON media.stickers
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM media.sticker_packs sp
            WHERE sp.id = media.stickers.sticker_pack_id
            AND sp.creator_user_id = auth.current_user_id()
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM media.sticker_packs sp
            WHERE sp.id = media.stickers.sticker_pack_id
            AND sp.creator_user_id = auth.current_user_id()
        )
    );

-- =====================================================
-- USER STICKER PACKS POLICIES
-- =====================================================

-- Users can manage their own installed sticker packs
CREATE POLICY user_sticker_packs_all_own
    ON media.user_sticker_packs
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- =====================================================
-- GIFS POLICIES
-- =====================================================

-- Everyone can view GIFs
CREATE POLICY gifs_select_all
    ON media.gifs
    FOR SELECT
    USING (TRUE);

-- Service role can manage GIFs
CREATE POLICY gifs_service_all
    ON media.gifs
    FOR ALL
    USING (TRUE);

-- =====================================================
-- FAVORITE GIFS POLICIES
-- =====================================================

-- Users can manage their own favorite GIFs
CREATE POLICY favorite_gifs_all_own
    ON media.favorite_gifs
    FOR ALL
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- =====================================================
-- STORAGE STATS POLICIES
-- =====================================================

-- Users can view their own storage stats
CREATE POLICY storage_stats_select_own
    ON media.storage_stats
    FOR SELECT
    USING (user_id = auth.current_user_id());

-- Service role can manage storage stats
CREATE POLICY storage_stats_service_all
    ON media.storage_stats
    FOR ALL
    USING (TRUE);