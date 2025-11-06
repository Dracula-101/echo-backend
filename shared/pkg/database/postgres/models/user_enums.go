package models

import (
	"database/sql/driver"
	"fmt"
)

// OnlineStatus represents the online status of a user
type OnlineStatus string

const (
	OnlineStatusOnline    OnlineStatus = "online"
	OnlineStatusOffline   OnlineStatus = "offline"
	OnlineStatusAway      OnlineStatus = "away"
	OnlineStatusBusy      OnlineStatus = "busy"
	OnlineStatusInvisible OnlineStatus = "invisible"
)

func (o OnlineStatus) IsValid() bool {
	switch o {
	case OnlineStatusOnline, OnlineStatusOffline, OnlineStatusAway, OnlineStatusBusy, OnlineStatusInvisible:
		return true
	}
	return false
}

func (o OnlineStatus) Value() (driver.Value, error) {
	if !o.IsValid() {
		return nil, fmt.Errorf("invalid online status: %s", o)
	}
	return string(o), nil
}

func (o *OnlineStatus) Scan(value interface{}) error {
	if value == nil {
		*o = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan OnlineStatus: expected string, got %T", value)
	}
	*o = OnlineStatus(str)
	if !o.IsValid() {
		return fmt.Errorf("invalid online status value: %s", str)
	}
	return nil
}

// ProfileVisibility represents the visibility level of a user profile
type ProfileVisibility string

const (
	ProfileVisibilityPublic  ProfileVisibility = "public"
	ProfileVisibilityFriends ProfileVisibility = "friends"
	ProfileVisibilityPrivate ProfileVisibility = "private"
)

func (p ProfileVisibility) IsValid() bool {
	switch p {
	case ProfileVisibilityPublic, ProfileVisibilityFriends, ProfileVisibilityPrivate:
		return true
	}
	return false
}

func (p ProfileVisibility) Value() (driver.Value, error) {
	if !p.IsValid() {
		return nil, fmt.Errorf("invalid profile visibility: %s", p)
	}
	return string(p), nil
}

func (p *ProfileVisibility) Scan(value interface{}) error {
	if value == nil {
		*p = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ProfileVisibility: expected string, got %T", value)
	}
	*p = ProfileVisibility(str)
	if !p.IsValid() {
		return fmt.Errorf("invalid profile visibility value: %s", str)
	}
	return nil
}

// RelationshipType represents the type of relationship between users
type RelationshipType string

const (
	RelationshipTypeFriend    RelationshipType = "friend"
	RelationshipTypeBlocked   RelationshipType = "blocked"
	RelationshipTypePending   RelationshipType = "pending"
	RelationshipTypeFollower  RelationshipType = "follower"
	RelationshipTypeFollowing RelationshipType = "following"
)

func (r RelationshipType) IsValid() bool {
	switch r {
	case RelationshipTypeFriend, RelationshipTypeBlocked, RelationshipTypePending,
		RelationshipTypeFollower, RelationshipTypeFollowing:
		return true
	}
	return false
}

func (r RelationshipType) Value() (driver.Value, error) {
	if !r.IsValid() {
		return nil, fmt.Errorf("invalid relationship type: %s", r)
	}
	return string(r), nil
}

func (r *RelationshipType) Scan(value interface{}) error {
	if value == nil {
		*r = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan RelationshipType: expected string, got %T", value)
	}
	*r = RelationshipType(str)
	if !r.IsValid() {
		return fmt.Errorf("invalid relationship type value: %s", str)
	}
	return nil
}

// ContactStatus represents the status of a contact relationship
type ContactStatus string

const (
	ContactStatusActive  ContactStatus = "active"
	ContactStatusPending ContactStatus = "pending"
	ContactStatusBlocked ContactStatus = "blocked"
	ContactStatusDeleted ContactStatus = "deleted"
)

func (c ContactStatus) IsValid() bool {
	switch c {
	case ContactStatusActive, ContactStatusPending, ContactStatusBlocked, ContactStatusDeleted:
		return true
	}
	return false
}

func (c ContactStatus) Value() (driver.Value, error) {
	if !c.IsValid() {
		return nil, fmt.Errorf("invalid contact status: %s", c)
	}
	return string(c), nil
}

func (c *ContactStatus) Scan(value interface{}) error {
	if value == nil {
		*c = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ContactStatus: expected string, got %T", value)
	}
	*c = ContactStatus(str)
	if !c.IsValid() {
		return fmt.Errorf("invalid contact status value: %s", str)
	}
	return nil
}

// BlockType represents the type of block
type BlockType string

const (
	BlockTypeUser   BlockType = "user"
	BlockTypeDomain BlockType = "domain"
	BlockTypeIP     BlockType = "ip"
)

func (b BlockType) IsValid() bool {
	switch b {
	case BlockTypeUser, BlockTypeDomain, BlockTypeIP:
		return true
	}
	return false
}

func (b BlockType) Value() (driver.Value, error) {
	if !b.IsValid() {
		return nil, fmt.Errorf("invalid block type: %s", b)
	}
	return string(b), nil
}

func (b *BlockType) Scan(value interface{}) error {
	if value == nil {
		*b = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan BlockType: expected string, got %T", value)
	}
	*b = BlockType(str)
	if !b.IsValid() {
		return fmt.Errorf("invalid block type value: %s", str)
	}
	return nil
}

// StatusPrivacy represents the privacy level of a status
type StatusPrivacy string

const (
	StatusPrivacyPublic   StatusPrivacy = "public"
	StatusPrivacyContacts StatusPrivacy = "contacts"
	StatusPrivacyPrivate  StatusPrivacy = "private"
)

func (s StatusPrivacy) IsValid() bool {
	switch s {
	case StatusPrivacyPublic, StatusPrivacyContacts, StatusPrivacyPrivate:
		return true
	}
	return false
}

func (s StatusPrivacy) Value() (driver.Value, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("invalid status privacy: %s", s)
	}
	return string(s), nil
}

func (s *StatusPrivacy) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan StatusPrivacy: expected string, got %T", value)
	}
	*s = StatusPrivacy(str)
	if !s.IsValid() {
		return fmt.Errorf("invalid status privacy value: %s", str)
	}
	return nil
}

// NotificationSound represents the notification sound setting
type NotificationSound string

const (
	NotificationSoundDefault NotificationSound = "default"
	NotificationSoundSilent  NotificationSound = "silent"
	NotificationSoundCustom  NotificationSound = "custom"
)

func (n NotificationSound) IsValid() bool {
	switch n {
	case NotificationSoundDefault, NotificationSoundSilent, NotificationSoundCustom:
		return true
	}
	return false
}

func (n NotificationSound) Value() (driver.Value, error) {
	if !n.IsValid() {
		return nil, fmt.Errorf("invalid notification sound: %s", n)
	}
	return string(n), nil
}

func (n *NotificationSound) Scan(value interface{}) error {
	if value == nil {
		*n = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan NotificationSound: expected string, got %T", value)
	}
	*n = NotificationSound(str)
	if !n.IsValid() {
		return fmt.Errorf("invalid notification sound value: %s", str)
	}
	return nil
}

// NotificationPreview represents the notification preview setting
type NotificationPreview string

const (
	NotificationPreviewFull NotificationPreview = "full"
	NotificationPreviewName NotificationPreview = "name"
	NotificationPreviewNone NotificationPreview = "none"
)

func (n NotificationPreview) IsValid() bool {
	switch n {
	case NotificationPreviewFull, NotificationPreviewName, NotificationPreviewNone:
		return true
	}
	return false
}

func (n NotificationPreview) Value() (driver.Value, error) {
	if !n.IsValid() {
		return nil, fmt.Errorf("invalid notification preview: %s", n)
	}
	return string(n), nil
}

func (n *NotificationPreview) Scan(value interface{}) error {
	if value == nil {
		*n = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan NotificationPreview: expected string, got %T", value)
	}
	*n = NotificationPreview(str)
	if !n.IsValid() {
		return fmt.Errorf("invalid notification preview value: %s", str)
	}
	return nil
}

// BackupFrequency represents the frequency of chat backups
type BackupFrequency string

const (
	BackupFrequencyDaily   BackupFrequency = "daily"
	BackupFrequencyWeekly  BackupFrequency = "weekly"
	BackupFrequencyMonthly BackupFrequency = "monthly"
	BackupFrequencyNever   BackupFrequency = "never"
)

func (b BackupFrequency) IsValid() bool {
	switch b {
	case BackupFrequencyDaily, BackupFrequencyWeekly, BackupFrequencyMonthly, BackupFrequencyNever:
		return true
	}
	return false
}

func (b BackupFrequency) Value() (driver.Value, error) {
	if !b.IsValid() {
		return nil, fmt.Errorf("invalid backup frequency: %s", b)
	}
	return string(b), nil
}

func (b *BackupFrequency) Scan(value interface{}) error {
	if value == nil {
		*b = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan BackupFrequency: expected string, got %T", value)
	}
	*b = BackupFrequency(str)
	if !b.IsValid() {
		return fmt.Errorf("invalid backup frequency value: %s", str)
	}
	return nil
}

// Theme represents the app theme setting
type Theme string

const (
	ThemeLight  Theme = "light"
	ThemeDark   Theme = "dark"
	ThemeSystem Theme = "system"
)

func (t Theme) IsValid() bool {
	switch t {
	case ThemeLight, ThemeDark, ThemeSystem:
		return true
	}
	return false
}

func (t Theme) Value() (driver.Value, error) {
	if !t.IsValid() {
		return nil, fmt.Errorf("invalid theme: %s", t)
	}
	return string(t), nil
}

func (t *Theme) Scan(value interface{}) error {
	if value == nil {
		*t = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan Theme: expected string, got %T", value)
	}
	*t = Theme(str)
	if !t.IsValid() {
		return fmt.Errorf("invalid theme value: %s", str)
	}
	return nil
}

// FontSize represents the font size setting
type FontSize string

const (
	FontSizeSmall  FontSize = "small"
	FontSizeMedium FontSize = "medium"
	FontSizeLarge  FontSize = "large"
)

func (f FontSize) IsValid() bool {
	switch f {
	case FontSizeSmall, FontSizeMedium, FontSizeLarge:
		return true
	}
	return false
}

func (f FontSize) Value() (driver.Value, error) {
	if !f.IsValid() {
		return nil, fmt.Errorf("invalid font size: %s", f)
	}
	return string(f), nil
}

func (f *FontSize) Scan(value interface{}) error {
	if value == nil {
		*f = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan FontSize: expected string, got %T", value)
	}
	*f = FontSize(str)
	if !f.IsValid() {
		return fmt.Errorf("invalid font size value: %s", str)
	}
	return nil
}

// DateFormat represents the date format setting
type DateFormat string

const (
	DateFormatDDMMYYYY DateFormat = "DD/MM/YYYY"
	DateFormatMMDDYYYY DateFormat = "MM/DD/YYYY"
	DateFormatYYYYMMDD DateFormat = "YYYY-MM-DD"
)

func (d DateFormat) IsValid() bool {
	switch d {
	case DateFormatDDMMYYYY, DateFormatMMDDYYYY, DateFormatYYYYMMDD:
		return true
	}
	return false
}

func (d DateFormat) Value() (driver.Value, error) {
	if !d.IsValid() {
		return nil, fmt.Errorf("invalid date format: %s", d)
	}
	return string(d), nil
}

func (d *DateFormat) Scan(value interface{}) error {
	if value == nil {
		*d = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan DateFormat: expected string, got %T", value)
	}
	*d = DateFormat(str)
	if !d.IsValid() {
		return fmt.Errorf("invalid date format value: %s", str)
	}
	return nil
}

// TimeFormat represents the time format setting
type TimeFormat string

const (
	TimeFormat12Hour TimeFormat = "12h"
	TimeFormat24Hour TimeFormat = "24h"
)

func (t TimeFormat) IsValid() bool {
	switch t {
	case TimeFormat12Hour, TimeFormat24Hour:
		return true
	}
	return false
}

func (t TimeFormat) Value() (driver.Value, error) {
	if !t.IsValid() {
		return nil, fmt.Errorf("invalid time format: %s", t)
	}
	return string(t), nil
}

func (t *TimeFormat) Scan(value interface{}) error {
	if value == nil {
		*t = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan TimeFormat: expected string, got %T", value)
	}
	*t = TimeFormat(str)
	if !t.IsValid() {
		return fmt.Errorf("invalid time format value: %s", str)
	}
	return nil
}

// UserReportStatus represents the status of a user report
type UserReportStatus string

const (
	UserReportStatusPending   UserReportStatus = "pending"
	UserReportStatusInReview  UserReportStatus = "in_review"
	UserReportStatusResolved  UserReportStatus = "resolved"
	UserReportStatusDismissed UserReportStatus = "dismissed"
)

func (u UserReportStatus) IsValid() bool {
	switch u {
	case UserReportStatusPending, UserReportStatusInReview, UserReportStatusResolved, UserReportStatusDismissed:
		return true
	}
	return false
}

func (u UserReportStatus) Value() (driver.Value, error) {
	if !u.IsValid() {
		return nil, fmt.Errorf("invalid user report status: %s", u)
	}
	return string(u), nil
}

func (u *UserReportStatus) Scan(value interface{}) error {
	if value == nil {
		*u = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan UserReportStatus: expected string, got %T", value)
	}
	*u = UserReportStatus(str)
	if !u.IsValid() {
		return fmt.Errorf("invalid user report status value: %s", str)
	}
	return nil
}

// UserReportPriority represents the priority level of a user report
type UserReportPriority string

const (
	UserReportPriorityLow      UserReportPriority = "low"
	UserReportPriorityMedium   UserReportPriority = "medium"
	UserReportPriorityHigh     UserReportPriority = "high"
	UserReportPriorityCritical UserReportPriority = "critical"
)

func (u UserReportPriority) IsValid() bool {
	switch u {
	case UserReportPriorityLow, UserReportPriorityMedium, UserReportPriorityHigh, UserReportPriorityCritical:
		return true
	}
	return false
}

func (u UserReportPriority) Value() (driver.Value, error) {
	if !u.IsValid() {
		return nil, fmt.Errorf("invalid user report priority: %s", u)
	}
	return string(u), nil
}

func (u *UserReportPriority) Scan(value interface{}) error {
	if value == nil {
		*u = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan UserReportPriority: expected string, got %T", value)
	}
	*u = UserReportPriority(str)
	if !u.IsValid() {
		return fmt.Errorf("invalid user report priority value: %s", str)
	}
	return nil
}
