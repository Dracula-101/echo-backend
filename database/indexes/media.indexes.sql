-- =====================================================
-- MEDIA SCHEMA - INDEXES
-- =====================================================

-- Files table indexes
CREATE INDEX idx_media_files_uploader ON media.files(uploader_user_id);
CREATE INDEX idx_media_files_type ON media.files(file_type);
CREATE INDEX idx_media_files_category ON media.files(file_category);
CREATE INDEX idx_media_files_created ON media.files(created_at);
CREATE INDEX idx_media_files_uploaded ON media.files(uploaded_at);
CREATE INDEX idx_media_files_hash ON media.files(content_hash) WHERE content_hash IS NOT NULL;
CREATE INDEX idx_media_files_checksum ON media.files(checksum) WHERE checksum IS NOT NULL;
CREATE INDEX idx_media_files_processing ON media.files(processing_status);
CREATE INDEX idx_media_files_moderation ON media.files(moderation_status);
CREATE INDEX idx_media_files_nsfw ON media.files(is_nsfw) WHERE is_nsfw = TRUE;
CREATE INDEX idx_media_files_visibility ON media.files(visibility);
CREATE INDEX idx_media_files_access_token ON media.files(access_token) WHERE access_token IS NOT NULL;
CREATE INDEX idx_media_files_expires ON media.files(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_media_files_deleted ON media.files(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_media_files_permanently_delete ON media.files(permanently_delete_at) WHERE permanently_delete_at IS NOT NULL;
CREATE INDEX idx_media_files_storage_key ON media.files(storage_key);
CREATE INDEX idx_media_files_mime_type ON media.files(mime_type);
CREATE INDEX idx_media_files_size ON media.files(file_size_bytes);

-- Processing queue table indexes
CREATE INDEX idx_media_processing_file ON media.processing_queue(file_id);
CREATE INDEX idx_media_processing_task ON media.processing_queue(task_type);
CREATE INDEX idx_media_processing_status ON media.processing_queue(status);
CREATE INDEX idx_media_processing_priority ON media.processing_queue(priority DESC, created_at);
CREATE INDEX idx_media_processing_worker ON media.processing_queue(worker_id) WHERE worker_id IS NOT NULL;
CREATE INDEX idx_media_processing_created ON media.processing_queue(created_at);
CREATE INDEX idx_media_processing_queued ON media.processing_queue(status, priority) WHERE status = 'queued';

-- Thumbnails table indexes
CREATE INDEX idx_media_thumbnails_file ON media.thumbnails(file_id);
CREATE INDEX idx_media_thumbnails_size ON media.thumbnails(size_type);
CREATE INDEX idx_media_thumbnails_created ON media.thumbnails(created_at);

-- Transcoding jobs table indexes
CREATE INDEX idx_media_transcoding_source ON media.transcoding_jobs(source_file_id);
CREATE INDEX idx_media_transcoding_output ON media.transcoding_jobs(output_file_id);
CREATE INDEX idx_media_transcoding_status ON media.transcoding_jobs(status);
CREATE INDEX idx_media_transcoding_profile ON media.transcoding_jobs(profile_name);
CREATE INDEX idx_media_transcoding_created ON media.transcoding_jobs(created_at);

-- Albums table indexes
CREATE INDEX idx_media_albums_user ON media.albums(user_id);
CREATE INDEX idx_media_albums_type ON media.albums(album_type);
CREATE INDEX idx_media_albums_system ON media.albums(is_system_album) WHERE is_system_album = TRUE;
CREATE INDEX idx_media_albums_visibility ON media.albums(visibility);
CREATE INDEX idx_media_albums_created ON media.albums(created_at);

-- Album files table indexes
CREATE INDEX idx_media_album_files_album ON media.album_files(album_id);
CREATE INDEX idx_media_album_files_file ON media.album_files(file_id);
CREATE INDEX idx_media_album_files_order ON media.album_files(album_id, display_order);
CREATE INDEX idx_media_album_files_added ON media.album_files(added_at);

-- Tags table indexes
CREATE INDEX idx_media_tags_user ON media.tags(user_id);
CREATE INDEX idx_media_tags_name ON media.tags(tag_name);
CREATE INDEX idx_media_tags_type ON media.tags(tag_type);
CREATE INDEX idx_media_tags_usage ON media.tags(usage_count DESC);

-- File tags table indexes
CREATE INDEX idx_media_file_tags_file ON media.file_tags(file_id);
CREATE INDEX idx_media_file_tags_tag ON media.file_tags(tag_id);
CREATE INDEX idx_media_file_tags_confidence ON media.file_tags(confidence_score) WHERE confidence_score IS NOT NULL;

-- Shares table indexes
CREATE INDEX idx_media_shares_file ON media.shares(file_id);
CREATE INDEX idx_media_shares_shared_by ON media.shares(shared_by_user_id);
CREATE INDEX idx_media_shares_shared_with_user ON media.shares(shared_with_user_id);
CREATE INDEX idx_media_shares_shared_with_conversation ON media.shares(shared_with_conversation_id);
CREATE INDEX idx_media_shares_token ON media.shares(share_token) WHERE share_token IS NOT NULL;
CREATE INDEX idx_media_shares_expires ON media.shares(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_media_shares_active ON media.shares(is_active) WHERE is_active = TRUE;

-- Access log table indexes
CREATE INDEX idx_media_access_log_file ON media.access_log(file_id);
CREATE INDEX idx_media_access_log_user ON media.access_log(user_id);
CREATE INDEX idx_media_access_log_type ON media.access_log(access_type);
CREATE INDEX idx_media_access_log_created ON media.access_log(created_at);
CREATE INDEX idx_media_access_log_ip ON media.access_log(ip_address);
CREATE INDEX idx_media_access_log_success ON media.access_log(success);

-- Sticker packs table indexes
CREATE INDEX idx_media_sticker_packs_creator ON media.sticker_packs(creator_user_id);
CREATE INDEX idx_media_sticker_packs_official ON media.sticker_packs(is_official) WHERE is_official = TRUE;
CREATE INDEX idx_media_sticker_packs_public ON media.sticker_packs(is_public) WHERE is_public = TRUE;
CREATE INDEX idx_media_sticker_packs_animated ON media.sticker_packs(is_animated) WHERE is_animated = TRUE;
CREATE INDEX idx_media_sticker_packs_downloads ON media.sticker_packs(download_count DESC);

-- Stickers table indexes
CREATE INDEX idx_media_stickers_pack ON media.stickers(sticker_pack_id);
CREATE INDEX idx_media_stickers_file ON media.stickers(file_id);
CREATE INDEX idx_media_stickers_creator ON media.stickers(creator_user_id);
CREATE INDEX idx_media_stickers_usage ON media.stickers(usage_count DESC);
CREATE INDEX idx_media_stickers_active ON media.stickers(is_active) WHERE is_active = TRUE;

-- User sticker packs table indexes
CREATE INDEX idx_media_user_sticker_packs_user ON media.user_sticker_packs(user_id);
CREATE INDEX idx_media_user_sticker_packs_pack ON media.user_sticker_packs(sticker_pack_id);
CREATE INDEX idx_media_user_sticker_packs_order ON media.user_sticker_packs(user_id, display_order);

-- GIFs table indexes
CREATE INDEX idx_media_gifs_provider ON media.gifs(provider, provider_gif_id);
CREATE INDEX idx_media_gifs_usage ON media.gifs(usage_count DESC);
CREATE INDEX idx_media_gifs_trending ON media.gifs(is_trending) WHERE is_trending = TRUE;
CREATE INDEX idx_media_gifs_tags ON media.gifs USING GIN(tags);

-- Favorite GIFs table indexes
CREATE INDEX idx_media_favorite_gifs_user ON media.favorite_gifs(user_id);
CREATE INDEX idx_media_favorite_gifs_gif ON media.favorite_gifs(gif_id);
CREATE INDEX idx_media_favorite_gifs_added ON media.favorite_gifs(added_at);

-- Storage stats table indexes
CREATE INDEX idx_media_storage_stats_user ON media.storage_stats(user_id);
CREATE INDEX idx_media_storage_stats_usage ON media.storage_stats(storage_used_percentage DESC);
CREATE INDEX idx_media_storage_stats_calculated ON media.storage_stats(last_calculated_at);