# Media Service Implementation Guide

## Overview

The media service is a comprehensive file management system that handles all types of media for the Echo messaging platform. It implements structured storage, intelligent file organization, and supports various use cases including profile photos, message attachments, albums, and file sharing.

## Architecture

### Structured Storage Paths

Files are organized in a hierarchical structure based on their purpose:

#### Profile Photos
```
{user_id}/profile/{hash}.{ext}
```
- Dedicated directory for profile photos
- Latest file is the active profile photo
- Easy to manage and update

#### Message Media
```
{user_id}/messages/{conversation_id}/{date}/{hash}.{ext}
```
- Organized by conversation for easy access
- Date-based folders for temporal organization
- Supports cleanup by conversation or date range

#### Albums
```
{user_id}/albums/{album_id}/{hash}.{ext}
```
- Custom album storage
- User-defined collections

#### Stickers
```
stickers/{pack_id}/{hash}.{ext}
```
- Global sticker storage
- Organized by sticker pack

#### Documents
```
{user_id}/documents/{date}/{hash}.{ext}
```
- General document storage
- Date-based organization

#### Voice Notes
```
{user_id}/voice/{conversation_id}/{date}/{hash}.{ext}
```
- Similar to message media but specifically for voice notes
- Organized by conversation

#### General Files
```
{user_id}/general/{date}/{hash}.{ext}
```
- Fallback for unspecified file types
- Date-based organization

## Key Features

### 1. File Context Types

Files are categorized by their usage context:
- `profile_photo` - User profile pictures
- `message_media` - Media sent in messages
- `album` - Album/collection media
- `sticker` - Sticker pack stickers
- `document` - Document files
- `voice_note` - Voice recordings
- `general` - General purpose files

### 2. Profile Photo Upload

**Endpoint**: POST `/api/v1/media/profile-photo`

**Features**:
- Maximum 10MB file size limit
- Automatic thumbnail generation (pending)
- Public visibility by default
- Structured storage path

**Usage**:
```go
output, err := mediaService.UploadProfilePhoto(ctx, UploadProfilePhotoInput{
    UserID: "user-uuid",
    FileReader: fileReader,
    FileName: "profile.jpg",
    FileSize: 1024000,
    ContentType: "image/jpeg",
})
```

### 3. Message Media Upload

**Endpoint**: POST `/api/v1/media/message`

**Features**:
- Organized by conversation
- Private visibility by default
- Supports captions
- Automatic thumbnail generation for images/videos

**Usage**:
```go
output, err := mediaService.UploadMessageMedia(ctx, UploadMessageMediaInput{
    UserID: "user-uuid",
    ConversationID: "conv-uuid",
    MessageID: "msg-uuid",
    FileReader: fileReader,
    FileName: "photo.jpg",
    FileSize: 2048000,
    ContentType: "image/jpeg",
    Caption: "Check this out!",
})
```

### 4. Album Management

Create albums to organize media:

```go
// Create album
album, err := mediaService.CreateAlbum(ctx, CreateAlbumInput{
    UserID: "user-uuid",
    Title: "Vacation 2024",
    Description: "Summer vacation photos",
    AlbumType: "custom",
    Visibility: "private",
})

// Add files to album
err = mediaService.AddFileToAlbum(ctx, AddFileToAlbumInput{
    AlbumID: album.AlbumID,
    FileID: "file-uuid",
    UserID: "user-uuid",
    DisplayOrder: 1,
})

// List albums
albums, err := mediaService.ListAlbums(ctx, ListAlbumsInput{
    UserID: "user-uuid",
    Limit: 20,
    Offset: 0,
})
```

### 5. File Sharing

Share files with expiration and access controls:

```go
// Create share
share, err := mediaService.CreateShare(ctx, CreateShareInput{
    FileID: "file-uuid",
    UserID: "user-uuid",
    SharedWithUser: "recipient-uuid", // optional
    ConversationID: "conv-uuid",      // optional
    AccessType: "view",               // view, download, edit
    ExpiresIn: &duration,             // optional
    MaxViews: &maxViews,              // optional
    Password: "secret123",            // optional
})

// Share URL: https://example.com/share/{share.ShareToken}

// Revoke share
err = mediaService.RevokeShare(ctx, RevokeShareInput{
    ShareID: share.ShareID,
    UserID: "user-uuid",
})
```

### 6. Storage Statistics

Track user storage usage:

```go
stats, err := mediaService.GetStorageStats(ctx, GetStorageStatsInput{
    UserID: "user-uuid",
})

// Returns:
// - Total files count
// - Total size in bytes and MB
// - Breakdown by category (images, videos, audio, documents)
// - Storage quota and usage percentage
```

## Database Schema

The service uses the following tables in the `media` schema:

- `media.files` - Main file storage table
- `media.albums` - Album definitions
- `media.album_files` - Many-to-many album-file relationships
- `media.shares` - File sharing records
- `media.access_log` - File access audit trail
- `media.storage_stats` - User storage statistics
- `media.processing_queue` - Async processing tasks (future)
- `media.thumbnails` - Thumbnail tracking (future)
- `media.sticker_packs` - Sticker pack management (future)
- `media.stickers` - Individual stickers (future)

## Repository Layer

Located in `internal/repo/file_repo.go`, implements:

### File Operations
- `CreateFile` - Create file record
- `GetFileByID` - Retrieve file metadata
- `ListFilesByUser` - List user's files
- `GetFileByContentHash` - Deduplication lookup
- `SoftDeleteFile` - Soft delete
- `HardDeleteFile` - Permanent delete
- `GetUserStorageUsage` - Calculate storage usage

### Album Operations
- `CreateAlbum` - Create new album
- `GetAlbumByID` - Retrieve album
- `ListAlbumsByUser` - List user's albums
- `UpdateAlbum` - Update album metadata
- `DeleteAlbum` - Delete album
- `AddFileToAlbum` - Add file to album
- `RemoveFileFromAlbum` - Remove file from album
- `ListAlbumFiles` - List files in album

### Share Operations
- `CreateShare` - Create file share
- `GetShareByID` - Retrieve share by ID
- `GetShareByToken` - Retrieve share by token
- `RevokeShare` - Revoke/disable share
- `IncrementShareViewCount` - Track views
- `IncrementShareDownloadCount` - Track downloads

### Statistics Operations
- `GetStorageStats` - Get cached statistics
- `CalculateStorageStats` - Recalculate from scratch
- `CreateOrUpdateStorageStats` - Upsert statistics

## Service Layer

Located in:
- `internal/service/media_service.go` - Core upload/download/delete
- `internal/service/media_service_extended.go` - Extended features
- `internal/service/storage_path.go` - Path generation utilities

### Storage Path Generation

The `GenerateStoragePath` function creates structured paths based on file context:

```go
storageKey := GenerateStoragePath(StoragePathConfig{
    UserID:         "user-uuid",
    ConversationID: "conv-uuid",  // optional
    AlbumID:        "album-uuid", // optional
    StickerPackID:  "pack-uuid",  // optional
    ContentHash:    "abc123...",
    FileExtension:  ".jpg",
    Context:        FileContextMessageMedia,
})
```

## Configuration

### Storage Settings

```yaml
storage:
  provider: r2              # r2, s3, local
  bucket: echo-media
  region: auto
  access_key_id: ${AWS_ACCESS_KEY_ID}
  secret_access_key: ${AWS_SECRET_ACCESS_KEY}
  endpoint: ${R2_ENDPOINT}
  public_url: ${PUBLIC_URL}
  use_cdn: true
  cdn_base_url: ${CDN_URL}
  max_file_size: 104857600  # 100MB
  upload_timeout: 5m
```

### Feature Flags

```yaml
features:
  deduplication:
    enabled: true
    algorithm: sha256

  thumbnails:
    enabled: true
    small_size:
      width: 150
      height: 150
    medium_size:
      width: 300
      height: 300
    large_size:
      width: 600
      height: 600
```

## Security Features

1. **Access Control**: Files have visibility levels (private, public, unlisted)
2. **Share Expiration**: Shares can expire after a time period
3. **View Limits**: Shares can have maximum view counts
4. **Password Protection**: Shares can be password protected
5. **Audit Logging**: All file accesses are logged
6. **Content Hashing**: SHA-256 for deduplication and integrity
7. **Storage Quotas**: Per-user storage limits

## Future Enhancements

### Planned Features

1. **Thumbnail Generation**
   - Automatic thumbnail creation for images/videos
   - Multiple sizes (small, medium, large)
   - WebP format support

2. **Video Transcoding**
   - Multiple quality levels (1080p, 720p, 480p)
   - Adaptive streaming support
   - Audio extraction

3. **Content Moderation**
   - NSFW detection
   - Automated content flagging
   - Manual moderation workflow

4. **Virus Scanning**
   - ClamAV integration
   - Quarantine infected files

5. **Advanced Search**
   - Full-text search
   - Tag-based filtering
   - AI-powered image recognition

6. **GIF Support**
   - Tenor/Giphy integration
   - Favorite GIFs
   - Custom GIF uploads

## API Endpoints

### File Operations
- `POST /api/v1/media/upload` - Generic file upload
- `GET /api/v1/media/files/:id` - Get file metadata
- `GET /api/v1/media/files/:id/download` - Download file
- `DELETE /api/v1/media/files/:id` - Delete file
- `GET /api/v1/media/files` - List user files

### Profile Photos
- `POST /api/v1/media/profile-photo` - Upload profile photo
- `GET /api/v1/media/profile-photo/:userId` - Get user's profile photo

### Message Media
- `POST /api/v1/media/message` - Upload message media
- `GET /api/v1/media/message/:messageId` - Get message media

### Albums
- `POST /api/v1/media/albums` - Create album
- `GET /api/v1/media/albums/:id` - Get album
- `GET /api/v1/media/albums` - List albums
- `PUT /api/v1/media/albums/:id` - Update album
- `DELETE /api/v1/media/albums/:id` - Delete album
- `POST /api/v1/media/albums/:id/files` - Add file to album
- `DELETE /api/v1/media/albums/:id/files/:fileId` - Remove file from album

### Sharing
- `POST /api/v1/media/shares` - Create share
- `GET /api/v1/media/share/:token` - Access shared file
- `DELETE /api/v1/media/shares/:id` - Revoke share

### Statistics
- `GET /api/v1/media/stats` - Get storage statistics

## Testing

### Unit Tests
```bash
cd services/media-service
go test -v ./...
```

### Integration Tests
```bash
cd services/media-service
go test -v -tags=integration ./...
```

## Monitoring

The service emits metrics for:
- Upload/download counts
- Storage usage
- Processing queue depth
- Error rates
- Response times

## Best Practices

1. **Always specify file context** when uploading to ensure proper path structure
2. **Use deduplication** for large files to save storage
3. **Set appropriate visibility** levels for files
4. **Implement thumbnail generation** for better UX
5. **Monitor storage quotas** to prevent abuse
6. **Use CDN URLs** for public files
7. **Implement retry logic** for uploads
8. **Validate file types** before upload
9. **Sanitize file names** to prevent path traversal
10. **Use presigned URLs** for direct client uploads (large files)

## Troubleshooting

### Common Issues

1. **Upload fails**: Check storage quota, file size limits
2. **Access denied**: Verify ownership and visibility settings
3. **File not found**: Check if soft-deleted, verify ID
4. **Slow uploads**: Consider enabling compression, use CDN
5. **Storage full**: Clean up old files, increase quota

## Examples

See `/examples` directory for complete working examples of:
- Profile photo upload flow
- Message media with thumbnails
- Album creation and management
- File sharing with expiration
- Storage statistics dashboard
