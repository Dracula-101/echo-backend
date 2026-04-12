CREATE SCHEMA IF NOT EXISTS media;

CREATE TABLE media.files (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uploader_user_id            UUID NOT NULL REFERENCES auth.users(id) ON DELETE SET NULL,
    file_name                   VARCHAR(500) NOT NULL,
    original_file_name          VARCHAR(500),
    file_type                   VARCHAR(100) NOT NULL,
    mime_type                   VARCHAR(255) NOT NULL,
    file_category               VARCHAR(50),
    file_extension              VARCHAR(20),
    file_size_bytes             BIGINT NOT NULL,
    storage_provider            VARCHAR(50) DEFAULT 'r2',
    storage_bucket              VARCHAR(255),
    storage_key                 TEXT NOT NULL,
    storage_url                 TEXT NOT NULL,
    storage_region              VARCHAR(100),
    cdn_url                     TEXT,
    has_thumbnail               BOOLEAN DEFAULT FALSE,
    thumbnail_url               TEXT,
    thumbnail_small_url         TEXT,
    thumbnail_medium_url        TEXT,
    thumbnail_large_url         TEXT,
    has_preview                 BOOLEAN DEFAULT FALSE,
    preview_url                 TEXT,
    width                       INTEGER,
    height                      INTEGER,
    duration_seconds            INTEGER,
    bitrate                     INTEGER,
    frame_rate                  DECIMAL(10,2),
    codec                       VARCHAR(100),
    resolution                  VARCHAR(50),
    aspect_ratio                VARCHAR(20),
    color_profile               VARCHAR(100),
    orientation                 INTEGER,
    has_alpha_channel           BOOLEAN,
    dominant_colors             TEXT[],
    video_codec                 VARCHAR(100),
    audio_codec                 VARCHAR(100),
    subtitle_tracks             JSONB DEFAULT '[]'::JSONB,
    audio_channels              INTEGER,
    sample_rate                 INTEGER,
    page_count                  INTEGER,
    word_count                  INTEGER,
    processing_status           VARCHAR(50) DEFAULT 'pending',
    processing_started_at       TIMESTAMPTZ,
    processing_completed_at     TIMESTAMPTZ,
    processing_error            TEXT,
    processing_attempts         INTEGER DEFAULT 0,
    is_encrypted                BOOLEAN DEFAULT FALSE,
    encryption_key_id           TEXT,
    content_hash                VARCHAR(255),
    checksum                    VARCHAR(255),
    is_scanned                  BOOLEAN DEFAULT FALSE,
    virus_scan_status           VARCHAR(50),
    virus_scan_at               TIMESTAMPTZ,
    moderation_status           VARCHAR(50) DEFAULT 'pending',
    moderation_score            DECIMAL(5,2),
    moderation_labels           JSONB DEFAULT '[]'::JSONB,
    is_nsfw                     BOOLEAN DEFAULT FALSE,
    nsfw_score                  DECIMAL(5,2),
    moderated_at                TIMESTAMPTZ,
    moderated_by_user_id        UUID REFERENCES auth.users(id),
    visibility                  VARCHAR(50) DEFAULT 'private',
    access_token                TEXT UNIQUE,
    expires_at                  TIMESTAMPTZ,
    max_downloads               INTEGER,
    download_count              INTEGER DEFAULT 0,
    view_count                  INTEGER DEFAULT 0,
    is_compressed               BOOLEAN DEFAULT FALSE,
    compression_ratio           DECIMAL(5,2),
    original_file_size_bytes    BIGINT,
    exif_data                   JSONB DEFAULT '{}'::JSONB,
    gps_latitude                DECIMAL(10,8),
    gps_longitude               DECIMAL(11,8),
    gps_altitude                DECIMAL(10,2),
    camera_make                 VARCHAR(255),
    camera_model                VARCHAR(255),
    lens_model                  VARCHAR(255),
    focal_length                DECIMAL(10,2),
    aperture                    DECIMAL(10,2),
    iso                         INTEGER,
    shutter_speed               VARCHAR(50),
    capture_date                TIMESTAMPTZ,
    last_accessed_at            TIMESTAMPTZ,
    access_count                BIGINT DEFAULT 0,
    uploaded_at                 TIMESTAMPTZ DEFAULT NOW(),
    created_at                  TIMESTAMPTZ DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ DEFAULT NOW(),
    deleted_at                  TIMESTAMPTZ,
    permanently_delete_at       TIMESTAMPTZ,
    uploaded_from_device_id     VARCHAR(255),
    uploaded_from_ip            INET,
    metadata                    JSONB DEFAULT '{}'::JSONB
);

CREATE TABLE media.processing_queue (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id         UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    task_type       VARCHAR(100) NOT NULL,
    priority        INTEGER DEFAULT 5,
    status          VARCHAR(50) DEFAULT 'queued',
    attempt_count   INTEGER DEFAULT 0,
    max_attempts    INTEGER DEFAULT 3,
    worker_id       VARCHAR(255),
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    error_message   TEXT,
    input_params    JSONB DEFAULT '{}'::JSONB,
    output_result   JSONB DEFAULT '{}'::JSONB,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE media.thumbnails (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id         UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    size_type       VARCHAR(50) NOT NULL,
    width           INTEGER NOT NULL,
    height          INTEGER NOT NULL,
    file_size_bytes BIGINT,
    storage_key     TEXT NOT NULL,
    storage_url     TEXT NOT NULL,
    format          VARCHAR(20),
    quality         INTEGER,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(file_id, size_type)
);

CREATE TABLE media.transcoding_jobs (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_file_id              UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    output_file_id              UUID REFERENCES media.files(id),
    profile_name                VARCHAR(100) NOT NULL,
    status                      VARCHAR(50) DEFAULT 'pending',
    progress_percentage         INTEGER DEFAULT 0,
    started_at                  TIMESTAMPTZ,
    completed_at                TIMESTAMPTZ,
    estimated_completion_at     TIMESTAMPTZ,
    error_message               TEXT,
    transcoding_params          JSONB DEFAULT '{}'::JSONB,
    created_at                  TIMESTAMPTZ DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE media.albums (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    cover_file_id   UUID REFERENCES media.files(id),
    album_type      VARCHAR(50) DEFAULT 'custom',
    is_system_album BOOLEAN DEFAULT FALSE,
    file_count      INTEGER DEFAULT 0,
    visibility      VARCHAR(50) DEFAULT 'private',
    sort_order      VARCHAR(50) DEFAULT 'date_desc',
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE media.album_files (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    album_id        UUID NOT NULL REFERENCES media.albums(id) ON DELETE CASCADE,
    file_id         UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    display_order   INTEGER,
    added_at        TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(album_id, file_id)
);

CREATE TABLE media.tags (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    tag_name    VARCHAR(100) NOT NULL,
    tag_type    VARCHAR(50) DEFAULT 'user',
    usage_count INTEGER DEFAULT 0,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, tag_name, tag_type)
);

CREATE TABLE media.file_tags (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id             UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    tag_id              UUID NOT NULL REFERENCES media.tags(id) ON DELETE CASCADE,
    confidence_score    DECIMAL(5,2),
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(file_id, tag_id)
);

CREATE TABLE media.shares (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id                     UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    shared_by_user_id           UUID NOT NULL REFERENCES auth.users(id),
    shared_with_user_id         UUID REFERENCES auth.users(id),
    shared_with_conversation_id UUID REFERENCES messages.conversations(id),
    share_token                 TEXT UNIQUE,
    access_type                 VARCHAR(50) DEFAULT 'view',
    password_hash               TEXT,
    expires_at                  TIMESTAMPTZ,
    max_views                   INTEGER,
    view_count                  INTEGER DEFAULT 0,
    download_count              INTEGER DEFAULT 0,
    is_active                   BOOLEAN DEFAULT TRUE,
    created_at                  TIMESTAMPTZ DEFAULT NOW(),
    revoked_at                  TIMESTAMPTZ
);

CREATE TABLE media.access_log (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id                 UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    user_id                 UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    access_type             VARCHAR(50) NOT NULL,
    ip_address              INET,
    user_agent              TEXT,
    device_id               VARCHAR(255),
    referrer                TEXT,
    success                 BOOLEAN DEFAULT TRUE,
    error_message           TEXT,
    bytes_transferred       BIGINT,
    access_duration_ms      INTEGER,
    created_at              TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE media.sticker_packs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_user_id     UUID REFERENCES auth.users(id),
    pack_name           VARCHAR(255) NOT NULL,
    pack_description    TEXT,
    cover_file_id       UUID REFERENCES media.files(id),
    icon_file_id        UUID REFERENCES media.files(id),
    sticker_count       INTEGER DEFAULT 0,
    is_official         BOOLEAN DEFAULT FALSE,
    is_animated         BOOLEAN DEFAULT FALSE,
    is_public           BOOLEAN DEFAULT FALSE,
    download_count      INTEGER DEFAULT 0,
    install_count       INTEGER DEFAULT 0,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE media.stickers (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_user_id     UUID REFERENCES auth.users(id),
    sticker_pack_id     UUID REFERENCES media.sticker_packs(id) ON DELETE CASCADE,
    file_id             UUID NOT NULL REFERENCES media.files(id) ON DELETE CASCADE,
    sticker_name        VARCHAR(255),
    emojis              TEXT[],
    is_animated         BOOLEAN DEFAULT FALSE,
    usage_count         BIGINT DEFAULT 0,
    is_active           BOOLEAN DEFAULT TRUE,
    created_at          TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE media.user_sticker_packs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    sticker_pack_id UUID NOT NULL REFERENCES media.sticker_packs(id) ON DELETE CASCADE,
    display_order   INTEGER,
    installed_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, sticker_pack_id)
);

CREATE TABLE media.gifs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider            VARCHAR(50),
    provider_gif_id     VARCHAR(255),
    title               TEXT,
    url                 TEXT NOT NULL,
    preview_url         TEXT,
    thumbnail_url       TEXT,
    width               INTEGER,
    height              INTEGER,
    file_size_bytes     BIGINT,
    tags                TEXT[],
    usage_count         BIGINT DEFAULT 0,
    is_trending         BOOLEAN DEFAULT FALSE,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(provider, provider_gif_id)
);

CREATE TABLE media.favorite_gifs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    gif_id      UUID NOT NULL REFERENCES media.gifs(id) ON DELETE CASCADE,
    added_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, gif_id)
);

CREATE TABLE media.storage_stats (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                     UUID UNIQUE NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    total_files                 INTEGER DEFAULT 0,
    total_size_bytes            BIGINT DEFAULT 0,
    images_count                INTEGER DEFAULT 0,
    images_size_bytes           BIGINT DEFAULT 0,
    videos_count                INTEGER DEFAULT 0,
    videos_size_bytes           BIGINT DEFAULT 0,
    audio_count                 INTEGER DEFAULT 0,
    audio_size_bytes            BIGINT DEFAULT 0,
    documents_count             INTEGER DEFAULT 0,
    documents_size_bytes        BIGINT DEFAULT 0,
    storage_quota_bytes         BIGINT DEFAULT 5368709120,
    storage_used_percentage     DECIMAL(5,2) DEFAULT 0.00,
    last_calculated_at          TIMESTAMPTZ DEFAULT NOW(),
    created_at                  TIMESTAMPTZ DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ DEFAULT NOW()
);
