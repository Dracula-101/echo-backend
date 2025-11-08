package models

import (
	"database/sql/driver"
	"fmt"
)

// FileProcessingStatus represents the processing status of a media file
type FileProcessingStatus string

const (
	FileProcessingStatusPending    FileProcessingStatus = "pending"
	FileProcessingStatusProcessing FileProcessingStatus = "processing"
	FileProcessingStatusCompleted  FileProcessingStatus = "completed"
	FileProcessingStatusFailed     FileProcessingStatus = "failed"
)

func (f FileProcessingStatus) IsValid() bool {
	switch f {
	case FileProcessingStatusPending, FileProcessingStatusProcessing,
		FileProcessingStatusCompleted, FileProcessingStatusFailed:
		return true
	}
	return false
}

func (f FileProcessingStatus) Value() (driver.Value, error) {
	if !f.IsValid() {
		return nil, fmt.Errorf("invalid file processing status: %s", f)
	}
	return string(f), nil
}

func (f *FileProcessingStatus) Scan(value interface{}) error {
	if value == nil {
		*f = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan FileProcessingStatus: expected string, got %T", value)
	}
	*f = FileProcessingStatus(str)
	if !f.IsValid() {
		return fmt.Errorf("invalid file processing status value: %s", str)
	}
	return nil
}

func (f FileProcessingStatus) String() string {
	return string(f)
}

// VirusScanStatus represents the virus scan status of a file
type VirusScanStatus string

const (
	VirusScanStatusClean      VirusScanStatus = "clean"
	VirusScanStatusInfected   VirusScanStatus = "infected"
	VirusScanStatusSuspicious VirusScanStatus = "suspicious"
	VirusScanStatusPending    VirusScanStatus = "pending"
)

func (v VirusScanStatus) IsValid() bool {
	switch v {
	case VirusScanStatusClean, VirusScanStatusInfected,
		VirusScanStatusSuspicious, VirusScanStatusPending:
		return true
	}
	return false
}

func (v VirusScanStatus) Value() (driver.Value, error) {
	if !v.IsValid() {
		return nil, fmt.Errorf("invalid virus scan status: %s", v)
	}
	return string(v), nil
}

func (v *VirusScanStatus) Scan(value interface{}) error {
	if value == nil {
		*v = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan VirusScanStatus: expected string, got %T", value)
	}
	*v = VirusScanStatus(str)
	if !v.IsValid() {
		return fmt.Errorf("invalid virus scan status value: %s", str)
	}
	return nil
}

// ModerationStatus represents the moderation status of content
type ModerationStatus string

const (
	ModerationStatusPending  ModerationStatus = "pending"
	ModerationStatusApproved ModerationStatus = "approved"
	ModerationStatusRejected ModerationStatus = "rejected"
	ModerationStatusFlagged  ModerationStatus = "flagged"
)

func (m ModerationStatus) IsValid() bool {
	switch m {
	case ModerationStatusPending, ModerationStatusApproved,
		ModerationStatusRejected, ModerationStatusFlagged:
		return true
	}
	return false
}

func (m ModerationStatus) Value() (driver.Value, error) {
	if !m.IsValid() {
		return nil, fmt.Errorf("invalid moderation status: %s", m)
	}
	return string(m), nil
}

func (m *ModerationStatus) Scan(value interface{}) error {
	if value == nil {
		*m = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ModerationStatus: expected string, got %T", value)
	}
	*m = ModerationStatus(str)
	if !m.IsValid() {
		return fmt.Errorf("invalid moderation status value: %s", str)
	}
	return nil
}

// MediaVisibility represents the visibility level of media
type MediaVisibility string

const (
	MediaVisibilityPrivate  MediaVisibility = "private"
	MediaVisibilityPublic   MediaVisibility = "public"
	MediaVisibilityUnlisted MediaVisibility = "unlisted"
)

func (m MediaVisibility) IsValid() bool {
	switch m {
	case MediaVisibilityPrivate, MediaVisibilityPublic, MediaVisibilityUnlisted:
		return true
	}
	return false
}

func (m MediaVisibility) Value() (driver.Value, error) {
	if !m.IsValid() {
		return nil, fmt.Errorf("invalid media visibility: %s", m)
	}
	return string(m), nil
}

func (m *MediaVisibility) Scan(value interface{}) error {
	if value == nil {
		*m = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan MediaVisibility: expected string, got %T", value)
	}
	*m = MediaVisibility(str)
	if !m.IsValid() {
		return fmt.Errorf("invalid media visibility value: %s", str)
	}
	return nil
}

func (m MediaVisibility) String() string {
	return string(m)
}

// AlbumType represents the type of media album
type AlbumType string

const (
	AlbumTypeCustom      AlbumType = "custom"
	AlbumTypeCameraRoll  AlbumType = "camera_roll"
	AlbumTypeScreenshots AlbumType = "screenshots"
	AlbumTypeFavorites   AlbumType = "favorites"
)

func (a AlbumType) IsValid() bool {
	switch a {
	case AlbumTypeCustom, AlbumTypeCameraRoll, AlbumTypeScreenshots, AlbumTypeFavorites:
		return true
	}
	return false
}

func (a AlbumType) Value() (driver.Value, error) {
	if !a.IsValid() {
		return nil, fmt.Errorf("invalid album type: %s", a)
	}
	return string(a), nil
}

func (a *AlbumType) Scan(value interface{}) error {
	if value == nil {
		*a = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan AlbumType: expected string, got %T", value)
	}
	*a = AlbumType(str)
	if !a.IsValid() {
		return fmt.Errorf("invalid album type value: %s", str)
	}
	return nil
}

// AlbumSortOrder represents the sort order of media in an album
type AlbumSortOrder string

const (
	AlbumSortOrderDateDesc AlbumSortOrder = "date_desc"
	AlbumSortOrderDateAsc  AlbumSortOrder = "date_asc"
	AlbumSortOrderName     AlbumSortOrder = "name"
	AlbumSortOrderManual   AlbumSortOrder = "manual"
)

func (a AlbumSortOrder) IsValid() bool {
	switch a {
	case AlbumSortOrderDateDesc, AlbumSortOrderDateAsc, AlbumSortOrderName, AlbumSortOrderManual:
		return true
	}
	return false
}

func (a AlbumSortOrder) Value() (driver.Value, error) {
	if !a.IsValid() {
		return nil, fmt.Errorf("invalid album sort order: %s", a)
	}
	return string(a), nil
}

func (a *AlbumSortOrder) Scan(value interface{}) error {
	if value == nil {
		*a = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan AlbumSortOrder: expected string, got %T", value)
	}
	*a = AlbumSortOrder(str)
	if !a.IsValid() {
		return fmt.Errorf("invalid album sort order value: %s", str)
	}
	return nil
}

// TagType represents the type of media tag
type TagType string

const (
	TagTypeUser        TagType = "user"
	TagTypeSystem      TagType = "system"
	TagTypeAIGenerated TagType = "ai_generated"
)

func (t TagType) IsValid() bool {
	switch t {
	case TagTypeUser, TagTypeSystem, TagTypeAIGenerated:
		return true
	}
	return false
}

func (t TagType) Value() (driver.Value, error) {
	if !t.IsValid() {
		return nil, fmt.Errorf("invalid tag type: %s", t)
	}
	return string(t), nil
}

func (t *TagType) Scan(value interface{}) error {
	if value == nil {
		*t = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan TagType: expected string, got %T", value)
	}
	*t = TagType(str)
	if !t.IsValid() {
		return fmt.Errorf("invalid tag type value: %s", str)
	}
	return nil
}

// ShareAccessType represents the access type for shared media
type ShareAccessType string

const (
	ShareAccessTypeView     ShareAccessType = "view"
	ShareAccessTypeDownload ShareAccessType = "download"
	ShareAccessTypeEdit     ShareAccessType = "edit"
)

func (s ShareAccessType) IsValid() bool {
	switch s {
	case ShareAccessTypeView, ShareAccessTypeDownload, ShareAccessTypeEdit:
		return true
	}
	return false
}

func (s ShareAccessType) Value() (driver.Value, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("invalid share access type: %s", s)
	}
	return string(s), nil
}

func (s *ShareAccessType) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ShareAccessType: expected string, got %T", value)
	}
	*s = ShareAccessType(str)
	if !s.IsValid() {
		return fmt.Errorf("invalid share access type value: %s", str)
	}
	return nil
}

// FileContext represents the context in which a file is used
type FileContext string

const (
	FileContextProfilePhoto  FileContext = "profile_photo"
	FileContextMessageMedia  FileContext = "message_media"
	FileContextAlbum         FileContext = "album"
	FileContextSticker       FileContext = "sticker"
	FileContextDocument      FileContext = "document"
	FileContextVoiceNote     FileContext = "voice_note"
	FileContextGeneral       FileContext = "general"
)

func (f FileContext) IsValid() bool {
	switch f {
	case FileContextProfilePhoto, FileContextMessageMedia, FileContextAlbum,
		FileContextSticker, FileContextDocument, FileContextVoiceNote, FileContextGeneral:
		return true
	}
	return false
}

func (f FileContext) Value() (driver.Value, error) {
	if !f.IsValid() {
		return nil, fmt.Errorf("invalid file context: %s", f)
	}
	return string(f), nil
}

func (f *FileContext) Scan(value interface{}) error {
	if value == nil {
		*f = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan FileContext: expected string, got %T", value)
	}
	*f = FileContext(str)
	if !f.IsValid() {
		return fmt.Errorf("invalid file context value: %s", str)
	}
	return nil
}

func (f FileContext) String() string {
	return string(f)
}
