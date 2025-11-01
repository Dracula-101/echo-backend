-- =====================================================
-- MEDIA SCHEMA - File Storage & Media Processing
-- =====================================================

-- Create Schema
CREATE SCHEMA IF NOT EXISTS media;

-- Media Files (R2 Storage references)
CREATE TABLE media.files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uploader_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE SET NULL,
    
    -- File details
    file_name VARCHAR(500) NOT NULL,
    original_file_name VARCHAR(500),
    file_type VARCHAR(100) NOT NULL, -- image/jpeg, video/mp4, audio/mpeg, application/pdf
    mime_type VARCHAR(255) NOT NULL,
    file_category VARCHAR(50), -- image, video, audio, document, archive
    file_extension VARCHAR(20),
    file_size_bytes BIGINT NOT NULL,
    
    -- Storage details
    storage_provider VARCHAR(50) DEFAULT 'r2', -- r2, s3, local
    storage_bucket VARCHAR(255),
    storage_key TEXT NOT NULL, -- R2 object key
    storage_url TEXT NOT NULL, -- Full URL to access file
    storage_region VARCHAR(100),
    cdn_url TEXT, -- CDN URL if available
    
    -- File variants (thumbnails, compressed versions)
    has_thumbnail BOOLEAN DEFAULT FALSE,
    thumbnail_url TEXT,
    thumbnail_small_url TEXT, -- 150x150
    thumbnail_medium_url TEXT, -- 300x300
    thumbnail_large_url TEXT, -- 600x600
    has_preview BOOLEAN DEFAULT FALSE,
    preview_url TEXT,
    
    -- Media-specific metadata
    width INTEGER,
    height INTEGER,
    duration_seconds INTEGER, -- For video/audio
    bitrate INTEGER,
    frame_rate DECIMAL(10,2),
    codec VARCHAR(100),
    resolution VARCHAR(50), -- 1080p, 4K, etc
    aspect_ratio VARCHAR(20),
    color_profile VARCHAR(100),
    orientation INTEGER, -- EXIF orientation
    
    -- Image specific
    has_alpha_channel BOOLEAN,
    dominant_colors TEXT[], -- Array of hex colors
    
    -- Video specific
    video_codec VARCHAR(100),
    audio_codec VARCHAR(100),
    subtitle_tracks JSONB DEFAULT '[]'::JSONB,
    
    -- Audio specific
    audio_channels INTEGER,
    sample_rate INTEGER,
    
    -- Document specific
    page_count INTEGER,
    word_count INTEGER,
    
    -- Processing status
    processing_status VARCHAR(50) DEFAULT 'pending', -- pending, processing, completed, failed
    processing_started_at TIMESTAMPTZ,
    processing_completed_at TIMESTAMPTZ,
    processing_error TEXT,
    processing_attempts INTEGER DEFAULT 0,
    
    -- Security & Content
    is_encrypted BOOLEAN DEFAULT FALSE,
    encryption_key_id TEXT,
    content_hash VARCHAR(255), -- SHA-256 hash for deduplication
    checksum VARCHAR(255), -- MD5 checksum
    is_scanned BOOLEAN DEFAULT FALSE,
    virus_scan_status VARCHAR(50), -- clean, infected, suspicious, pending
    virus_scan_at TIMESTAMPTZ,
    
    -- Content moderation
    moderation_status VARCHAR(50) DEFAULT 'pending', -- pending, approved, rejected, flagged
    moderation_score DECIMAL(5,2), -- 0-100
    moderation_labels JSONB DEFAULT '[]'::JSONB, -- AI-detected labels
    is_nsfw BOOLEAN DEFAULT FALSE,
    nsfw_score DECIMAL(5,2),
    moderated_at TIMESTAMPTZ,
    moderated_by_user_id UUID REFERENCES auth.users(id),
    
    -- Access control
    visibility VARCHAR(50) DEFAULT 'private', -- private, public, unlisted
    access_token TEXT UNIQUE,
    expires_at TIMESTAMPTZ,
    max_downloads INTEGER,
    download_count INTEGER DEFAULT 0,
    view_count INTEGER DEFAULT 0,
    
    -- Compression
    is_compressed BOOLEAN DEFAULT FALSE,
    compression_ratio DECIMAL(5,2),
    original_file_size_bytes BIGINT,
    
    -- EXIF & Metadata
    exif_data JSONB DEFAULT '{}'::JSONB,
    gps_latitude DECIMAL(10, 8),
    gps_longitude DECIMAL(11, 8),
    gps_altitude DECIMAL(10, 2),
    camera_make VARCHAR(255),
    camera_model VARCHAR(255),
    lens_model VARCHAR(255),
    focal_length DECIMAL(10,2),
    aperture DECIMAL(10,2),
    iso INTEGER,
    shutter_speed VARCHAR(50),
    capture_date TIMESTAMPTZ,
    
    -- Usage tracking
    last_accessed_at TIMESTAMPTZ,
    access_count BIGINT DEFAULT 0,
    
    -- Lifecycle
    uploaded_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    permanently_delete_at TIMESTAMPTZ, -- Soft delete with scheduled purge
    
    -- Device info
    uploaded_from_device_id VARCHAR(255),
    uploaded_from_ip INET,
    
    metadata JSONB DEFAULT '{}'::JSONB
);

-- Media Processing Queue
CREATE TABLE media.processing_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    task_type VARCHAR(100) NOT NULL, -- thumbnail, compress, transcode, scan, moderate
    priority INTEGER DEFAULT 5, -- 1-10, higher is more priority
    status VARCHAR(50) DEFAULT 'queued', -- queued, processing, completed, failed
    attempt_count INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    worker_id VARCHAR(255),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    input_params JSONB DEFAULT '{}'::JSONB,
    output_result JSONB DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Media Thumbnails (detailed tracking)
CREATE TABLE media.thumbnails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    size_type VARCHAR(50) NOT NULL, -- small, medium, large, custom
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    file_size_bytes BIGINT,
    storage_key TEXT NOT NULL,
    storage_url TEXT NOT NULL,
    format VARCHAR(20), -- jpeg, png, webp
    quality INTEGER, -- 1-100
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(file_id, size_type)
);

-- Media Transcoding Jobs (for video/audio)
CREATE TABLE media.transcoding_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    output_file_id UUID REFERENCES media.files(id),
    profile_name VARCHAR(100) NOT NULL, -- 1080p, 720p, 480p, audio_only
    status VARCHAR(50) DEFAULT 'pending',
    progress_percentage INTEGER DEFAULT 0,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    estimated_completion_at TIMESTAMPTZ,
    error_message TEXT,
    transcoding_params JSONB DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Media Albums/Collections
CREATE TABLE media.albums (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    cover_file_id UUID REFERENCES media.files(id),
    album_type VARCHAR(50) DEFAULT 'custom', -- custom, camera_roll, screenshots, favorites
    is_system_album BOOLEAN DEFAULT FALSE,
    file_count INTEGER DEFAULT 0,
    visibility VARCHAR(50) DEFAULT 'private',
    sort_order VARCHAR(50) DEFAULT 'date_desc', -- date_desc, date_asc, name, manual
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Media Album Files (many-to-many)
CREATE TABLE media.album_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    album_id UUID NOT NULL REFERENCES media.albums(id) ON DELETE CASCADE,
    file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    display_order INTEGER,
    added_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(album_id, file_id)
);

-- Media Tags
CREATE TABLE media.tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    tag_name VARCHAR(100) NOT NULL,
    tag_type VARCHAR(50) DEFAULT 'user', -- user, system, ai_generated
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, tag_name, tag_type)
);

-- Media File Tags (many-to-many)
CREATE TABLE media.file_tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES media.tags(id) ON DELETE CASCADE,
    confidence_score DECIMAL(5,2), -- For AI-generated tags
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(file_id, tag_id)
);

-- Media Shares (sharing files with specific users/groups)
CREATE TABLE media.shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    shared_by_user_id UUID NOT NULL REFERENCES auth.users(id),
    shared_with_user_id UUID REFERENCES auth.users(id),
    shared_with_conversation_id UUID REFERENCES messages.conversations(id),
    share_token TEXT UNIQUE,
    access_type VARCHAR(50) DEFAULT 'view', -- view, download, edit
    password_hash TEXT,
    expires_at TIMESTAMPTZ,
    max_views INTEGER,
    view_count INTEGER DEFAULT 0,
    download_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

-- Media Access Log
CREATE TABLE media.access_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    access_type VARCHAR(50) NOT NULL, -- view, download, upload, delete, share
    ip_address INET,
    user_agent TEXT,
    device_id VARCHAR(255),
    referrer TEXT,
    success BOOLEAN DEFAULT TRUE,
    error_message TEXT,
    bytes_transferred BIGINT,
    access_duration_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Sticker Packs (must be created before stickers table)
CREATE TABLE media.sticker_packs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_user_id UUID REFERENCES auth.users(id),
    pack_name VARCHAR(255) NOT NULL,
    pack_description TEXT,
    cover_file_id UUID REFERENCES media.files(id),
    icon_file_id UUID REFERENCES media.files(id),
    sticker_count INTEGER DEFAULT 0,
    is_official BOOLEAN DEFAULT FALSE,
    is_animated BOOLEAN DEFAULT FALSE,
    is_public BOOLEAN DEFAULT FALSE,
    download_count INTEGER DEFAULT 0,
    install_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Stickers
CREATE TABLE media.stickers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_user_id UUID REFERENCES auth.users(id),
    sticker_pack_id UUID REFERENCES media.sticker_packs(id) ON DELETE CASCADE,
    file_id UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    sticker_name VARCHAR(255),
    emojis TEXT[], -- Related emojis
    is_animated BOOLEAN DEFAULT FALSE,
    usage_count BIGINT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- User Installed Sticker Packs
CREATE TABLE media.user_sticker_packs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    sticker_pack_id UUID NOT NULL REFERENCES media.sticker_packs(id) ON DELETE CASCADE,
    display_order INTEGER,
    installed_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, sticker_pack_id)
);

-- GIFs (can reference Tenor/Giphy or stored)
CREATE TABLE media.gifs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50), -- tenor, giphy, custom
    provider_gif_id VARCHAR(255),
    title TEXT,
    url TEXT NOT NULL,
    preview_url TEXT,
    thumbnail_url TEXT,
    width INTEGER,
    height INTEGER,
    file_size_bytes BIGINT,
    tags TEXT[],
    usage_count BIGINT DEFAULT 0,
    is_trending BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(provider, provider_gif_id)
);

-- User Favorite GIFs
CREATE TABLE media.favorite_gifs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    gif_id UUID NOT NULL REFERENCES media.gifs(id) ON DELETE CASCADE,
    added_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, gif_id)
);

-- Media Storage Statistics (per user)
CREATE TABLE media.storage_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    total_files INTEGER DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,
    images_count INTEGER DEFAULT 0,
    images_size_bytes BIGINT DEFAULT 0,
    videos_count INTEGER DEFAULT 0,
    videos_size_bytes BIGINT DEFAULT 0,
    audio_count INTEGER DEFAULT 0,
    audio_size_bytes BIGINT DEFAULT 0,
    documents_count INTEGER DEFAULT 0,
    documents_size_bytes BIGINT DEFAULT 0,
    storage_quota_bytes BIGINT DEFAULT 5368709120, -- 5GB default
    storage_used_percentage DECIMAL(5,2) DEFAULT 0.00,
    last_calculated_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_files_uploader ON media.files(uploader_user_id);
CREATE INDEX idx_files_type ON media.files(file_category);
CREATE INDEX idx_files_created ON media.files(created_at);
CREATE INDEX idx_files_hash ON media.files(content_hash);
CREATE INDEX idx_files_status ON media.files(processing_status);
CREATE INDEX idx_processing_queue_status ON media.processing_queue(status, priority);
CREATE INDEX idx_albums_user ON media.albums(user_id);
CREATE INDEX idx_album_files_album ON media.album_files(album_id);
CREATE INDEX idx_file_tags_file ON media.file_tags(file_id);
CREATE INDEX idx_shares_file ON media.shares(file_id);
CREATE INDEX idx_access_log_file ON media.access_log(file_id);
CREATE INDEX idx_access_log_created ON media.access_log(created_at);