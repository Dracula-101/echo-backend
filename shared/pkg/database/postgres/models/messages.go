package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

type Conversation struct {
	ID                   string          `db:"id" json:"id" pk:"true"`
	ConversationType     string          `db:"conversation_type" json:"conversation_type"`
	Title                *string         `db:"title" json:"title,omitempty"`
	Description          *string         `db:"description" json:"description,omitempty"`
	AvatarURL            *string         `db:"avatar_url" json:"avatar_url,omitempty"`
	CreatorUserID        string          `db:"creator_user_id" json:"creator_user_id"`
	IsGroup              bool            `db:"is_group" json:"is_group"`
	IsChannel            bool            `db:"is_channel" json:"is_channel"`
	IsEncrypted          bool            `db:"is_encrypted" json:"is_encrypted"`
	EncryptionKeyID      *string         `db:"encryption_key_id" json:"encryption_key_id,omitempty"`
	MaxMembers           *int            `db:"max_members" json:"max_members,omitempty"`
	IsPublic             bool            `db:"is_public" json:"is_public"`
	InviteLink           *string         `db:"invite_link" json:"invite_link,omitempty"`
	InviteLinkExpiresAt  *time.Time      `db:"invite_link_expires_at" json:"invite_link_expires_at,omitempty"`
	JoinApprovalRequired bool            `db:"join_approval_required" json:"join_approval_required"`
	WhoCanSendMessages   string          `db:"who_can_send_messages" json:"who_can_send_messages"`
	WhoCanAddMembers     string          `db:"who_can_add_members" json:"who_can_add_members"`
	WhoCanEditInfo       string          `db:"who_can_edit_info" json:"who_can_edit_info"`
	WhoCanPinMessages    string          `db:"who_can_pin_messages" json:"who_can_pin_messages"`
	IsActive             bool            `db:"is_active" json:"is_active"`
	IsArchived           bool            `db:"is_archived" json:"is_archived"`
	ArchivedAt           *time.Time      `db:"archived_at" json:"archived_at,omitempty"`
	MemberCount          int             `db:"member_count" json:"member_count"`
	MessageCount         int64           `db:"message_count" json:"message_count"`
	LastMessageID        *string         `db:"last_message_id" json:"last_message_id,omitempty"`
	LastMessageAt        *time.Time      `db:"last_message_at" json:"last_message_at,omitempty"`
	LastActivityAt       time.Time       `db:"last_activity_at" json:"last_activity_at"`
	CreatedAt            time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt            time.Time       `db:"updated_at" json:"updated_at"`
	DeletedAt            *time.Time      `db:"deleted_at" json:"deleted_at,omitempty"`
	Metadata             json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

func (c *Conversation) TableName() string {
	return "messages.conversations"
}

func (c *Conversation) PrimaryKey() interface{} {
	return c.ID
}

type ConversationParticipant struct {
	ID                  string     `db:"id" json:"id" pk:"true"`
	ConversationID      string     `db:"conversation_id" json:"conversation_id"`
	UserID              string     `db:"user_id" json:"user_id"`
	Role                string     `db:"role" json:"role"`
	Nickname            *string    `db:"nickname" json:"nickname,omitempty"`
	CustomNotifications bool       `db:"custom_notifications" json:"custom_notifications"`
	IsMuted             bool       `db:"is_muted" json:"is_muted"`
	MutedUntil          *time.Time `db:"muted_until" json:"muted_until,omitempty"`
	IsPinned            bool       `db:"is_pinned" json:"is_pinned"`
	PinOrder            *int       `db:"pin_order" json:"pin_order,omitempty"`
	IsArchived          bool       `db:"is_archived" json:"is_archived"`
	LastReadMessageID   *string    `db:"last_read_message_id" json:"last_read_message_id,omitempty"`
	LastReadAt          *time.Time `db:"last_read_at" json:"last_read_at,omitempty"`
	UnreadCount         int        `db:"unread_count" json:"unread_count"`
	MentionCount        int        `db:"mention_count" json:"mention_count"`
	CanSendMessages     bool       `db:"can_send_messages" json:"can_send_messages"`
	CanSendMedia        bool       `db:"can_send_media" json:"can_send_media"`
	CanAddMembers       bool       `db:"can_add_members" json:"can_add_members"`
	CanRemoveMembers    bool       `db:"can_remove_members" json:"can_remove_members"`
	CanEditInfo         bool       `db:"can_edit_info" json:"can_edit_info"`
	CanPinMessages      bool       `db:"can_pin_messages" json:"can_pin_messages"`
	CanDeleteMessages   bool       `db:"can_delete_messages" json:"can_delete_messages"`
	JoinMethod          *string    `db:"join_method" json:"join_method,omitempty"`
	InvitedByUserID     *string    `db:"invited_by_user_id" json:"invited_by_user_id,omitempty"`
	JoinedAt            time.Time  `db:"joined_at" json:"joined_at"`
	LeftAt              *time.Time `db:"left_at" json:"left_at,omitempty"`
	RemovedAt           *time.Time `db:"removed_at" json:"removed_at,omitempty"`
	RemovedByUserID     *string    `db:"removed_by_user_id" json:"removed_by_user_id,omitempty"`
	RemovalReason       *string    `db:"removal_reason" json:"removal_reason,omitempty"`
	CreatedAt           time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at" json:"updated_at"`
}

func (c *ConversationParticipant) TableName() string {
	return "messages.conversation_participants"
}

func (c *ConversationParticipant) PrimaryKey() interface{} {
	return c.ID
}

type Message struct {
	ID                     string          `db:"id" json:"id" pk:"true"`
	ConversationID         string          `db:"conversation_id" json:"conversation_id"`
	SenderUserID           string          `db:"sender_user_id" json:"sender_user_id"`
	ParentMessageID        *string         `db:"parent_message_id" json:"parent_message_id,omitempty"`
	MessageType            string          `db:"message_type" json:"message_type"`
	Content                *string         `db:"content" json:"content,omitempty"`
	ContentEncrypted       bool            `db:"content_encrypted" json:"content_encrypted"`
	ContentHash            *string         `db:"content_hash" json:"content_hash,omitempty"`
	FormatType             string          `db:"format_type" json:"format_type"`
	Mentions               json.RawMessage `db:"mentions" json:"mentions,omitempty"`
	Hashtags               pq.StringArray  `db:"hashtags" json:"hashtags,omitempty"`
	Links                  json.RawMessage `db:"links" json:"links,omitempty"`
	Status                 string          `db:"status" json:"status"`
	IsEdited               bool            `db:"is_edited" json:"is_edited"`
	EditedAt               *time.Time      `db:"edited_at" json:"edited_at,omitempty"`
	EditHistory            json.RawMessage `db:"edit_history" json:"edit_history,omitempty"`
	IsDeleted              bool            `db:"is_deleted" json:"is_deleted"`
	DeletedAt              *time.Time      `db:"deleted_at" json:"deleted_at,omitempty"`
	DeletedFor             *string         `db:"deleted_for" json:"deleted_for,omitempty"`
	IsPinned               bool            `db:"is_pinned" json:"is_pinned"`
	PinnedAt               *time.Time      `db:"pinned_at" json:"pinned_at,omitempty"`
	PinnedByUserID         *string         `db:"pinned_by_user_id" json:"pinned_by_user_id,omitempty"`
	DeliveredAt            *time.Time      `db:"delivered_at" json:"delivered_at,omitempty"`
	DeliveryCount          int             `db:"delivery_count" json:"delivery_count"`
	ReadCount              int             `db:"read_count" json:"read_count"`
	IsFlagged              bool            `db:"is_flagged" json:"is_flagged"`
	FlagReason             *string         `db:"flag_reason" json:"flag_reason,omitempty"`
	FlaggedAt              *time.Time      `db:"flagged_at" json:"flagged_at,omitempty"`
	FlaggedByUserID        *string         `db:"flagged_by_user_id" json:"flagged_by_user_id,omitempty"`
	ScheduledAt            *time.Time      `db:"scheduled_at" json:"scheduled_at,omitempty"`
	IsScheduled            bool            `db:"is_scheduled" json:"is_scheduled"`
	ReplyCount             int             `db:"reply_count" json:"reply_count"`
	LastReplyAt            *time.Time      `db:"last_reply_at" json:"last_reply_at,omitempty"`
	ReactionCount          int             `db:"reaction_count" json:"reaction_count"`
	IsForwarded            bool            `db:"is_forwarded" json:"is_forwarded"`
	ForwardedFromMessageID *string         `db:"forwarded_from_message_id" json:"forwarded_from_message_id,omitempty"`
	ForwardCount           int             `db:"forward_count" json:"forward_count"`
	SentFromDeviceID       *string         `db:"sent_from_device_id" json:"sent_from_device_id,omitempty"`
	SentFromIP             *string         `db:"sent_from_ip" json:"sent_from_ip,omitempty"`
	ExpiresAt              *time.Time      `db:"expires_at" json:"expires_at,omitempty"`
	ExpireAfterSeconds     *int            `db:"expire_after_seconds" json:"expire_after_seconds,omitempty"`
	CreatedAt              time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time       `db:"updated_at" json:"updated_at"`
	Metadata               json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

func (m *Message) TableName() string {
	return "messages.messages"
}

func (m *Message) PrimaryKey() interface{} {
	return m.ID
}

type Reaction struct {
	ID               string    `db:"id" json:"id" pk:"true"`
	MessageID        string    `db:"message_id" json:"message_id"`
	UserID           string    `db:"user_id" json:"user_id"`
	ReactionType     string    `db:"reaction_type" json:"reaction_type"`
	ReactionEmoji    *string   `db:"reaction_emoji" json:"reaction_emoji,omitempty"`
	ReactionSkinTone *string   `db:"reaction_skin_tone" json:"reaction_skin_tone,omitempty"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

func (r *Reaction) TableName() string {
	return "messages.reactions"
}

func (r *Reaction) PrimaryKey() interface{} {
	return r.ID
}

type DeliveryStatus struct {
	ID           string     `db:"id" json:"id" pk:"true"`
	MessageID    string     `db:"message_id" json:"message_id"`
	UserID       string     `db:"user_id" json:"user_id"`
	Status       string     `db:"status" json:"status"`
	DeliveredAt  *time.Time `db:"delivered_at" json:"delivered_at,omitempty"`
	ReadAt       *time.Time `db:"read_at" json:"read_at,omitempty"`
	FailedReason *string    `db:"failed_reason" json:"failed_reason,omitempty"`
	RetryCount   int        `db:"retry_count" json:"retry_count"`
	DeviceID     *string    `db:"device_id" json:"device_id,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}

func (d *DeliveryStatus) TableName() string {
	return "messages.delivery_status"
}

func (d *DeliveryStatus) PrimaryKey() interface{} {
	return d.ID
}

type MessageMedia struct {
	ID           string    `db:"id" json:"id" pk:"true"`
	MessageID    string    `db:"message_id" json:"message_id"`
	MediaID      string    `db:"media_id" json:"media_id"`
	MediaType    string    `db:"media_type" json:"media_type"`
	DisplayOrder int       `db:"display_order" json:"display_order"`
	Caption      *string   `db:"caption" json:"caption,omitempty"`
	ThumbnailURL *string   `db:"thumbnail_url" json:"thumbnail_url,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

func (m *MessageMedia) TableName() string {
	return "messages.message_media"
}

func (m *MessageMedia) PrimaryKey() interface{} {
	return m.ID
}

type LinkPreview struct {
	ID          string    `db:"id" json:"id" pk:"true"`
	MessageID   string    `db:"message_id" json:"message_id"`
	URL         string    `db:"url" json:"url"`
	Title       *string   `db:"title" json:"title,omitempty"`
	Description *string   `db:"description" json:"description,omitempty"`
	ImageURL    *string   `db:"image_url" json:"image_url,omitempty"`
	FaviconURL  *string   `db:"favicon_url" json:"favicon_url,omitempty"`
	SiteName    *string   `db:"site_name" json:"site_name,omitempty"`
	ContentType *string   `db:"content_type" json:"content_type,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

func (l *LinkPreview) TableName() string {
	return "messages.link_previews"
}

func (l *LinkPreview) PrimaryKey() interface{} {
	return l.ID
}

type Poll struct {
	ID              string     `db:"id" json:"id" pk:"true"`
	MessageID       string     `db:"message_id" json:"message_id"`
	Question        string     `db:"question" json:"question"`
	AllowMultiple   bool       `db:"allow_multiple_answers" json:"allow_multiple_answers"`
	IsAnonymous     bool       `db:"is_anonymous" json:"is_anonymous"`
	IsQuiz          bool       `db:"is_quiz" json:"is_quiz"`
	CorrectOptionID *int       `db:"correct_option_id" json:"correct_option_id,omitempty"`
	Explanation     *string    `db:"explanation" json:"explanation,omitempty"`
	ClosesAt        *time.Time `db:"closes_at" json:"closes_at,omitempty"`
	IsClosed        bool       `db:"is_closed" json:"is_closed"`
	ClosedAt        *time.Time `db:"closed_at" json:"closed_at,omitempty"`
	TotalVotes      int        `db:"total_votes" json:"total_votes"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
}

func (p *Poll) TableName() string {
	return "messages.polls"
}

func (p *Poll) PrimaryKey() interface{} {
	return p.ID
}

type PollOption struct {
	ID             string    `db:"id" json:"id" pk:"true"`
	PollID         string    `db:"poll_id" json:"poll_id"`
	OptionText     string    `db:"option_text" json:"option_text"`
	OptionOrder    int       `db:"option_order" json:"option_order"`
	VoteCount      int       `db:"vote_count" json:"vote_count"`
	VotePercentage float64   `db:"vote_percentage" json:"vote_percentage"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

func (p *PollOption) TableName() string {
	return "messages.poll_options"
}

func (p *PollOption) PrimaryKey() interface{} {
	return p.ID
}

type PollVote struct {
	ID           string    `db:"id" json:"id" pk:"true"`
	PollID       string    `db:"poll_id" json:"poll_id"`
	PollOptionID string    `db:"poll_option_id" json:"poll_option_id"`
	UserID       string    `db:"user_id" json:"user_id"`
	VotedAt      time.Time `db:"voted_at" json:"voted_at"`
}

func (p *PollVote) TableName() string {
	return "messages.poll_votes"
}

func (p *PollVote) PrimaryKey() interface{} {
	return p.ID
}

type TypingIndicator struct {
	ID             string    `db:"id" json:"id" pk:"true"`
	ConversationID string    `db:"conversation_id" json:"conversation_id"`
	UserID         string    `db:"user_id" json:"user_id"`
	StartedAt      time.Time `db:"started_at" json:"started_at"`
	ExpiresAt      time.Time `db:"expires_at" json:"expires_at"`
}

func (t *TypingIndicator) TableName() string {
	return "messages.typing_indicators"
}

func (t *TypingIndicator) PrimaryKey() interface{} {
	return t.ID
}

type MessageReport struct {
	ID             string     `db:"id" json:"id" pk:"true"`
	MessageID      string     `db:"message_id" json:"message_id"`
	ReporterUserID string     `db:"reporter_user_id" json:"reporter_user_id"`
	ReportType     string     `db:"report_type" json:"report_type"`
	ReportCategory *string    `db:"report_category" json:"report_category,omitempty"`
	Description    *string    `db:"description" json:"description,omitempty"`
	Status         string     `db:"status" json:"status"`
	Priority       string     `db:"priority" json:"priority"`
	AssignedTo     *string    `db:"assigned_to" json:"assigned_to,omitempty"`
	Resolution     *string    `db:"resolution" json:"resolution,omitempty"`
	ActionTaken    *string    `db:"action_taken" json:"action_taken,omitempty"`
	ResolvedAt     *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
}

func (m *MessageReport) TableName() string {
	return "messages.message_reports"
}

func (m *MessageReport) PrimaryKey() interface{} {
	return m.ID
}

type Draft struct {
	ID               string          `db:"id" json:"id" pk:"true"`
	UserID           string          `db:"user_id" json:"user_id"`
	ConversationID   string          `db:"conversation_id" json:"conversation_id"`
	Content          *string         `db:"content" json:"content,omitempty"`
	ReplyToMessageID *string         `db:"reply_to_message_id" json:"reply_to_message_id,omitempty"`
	Mentions         json.RawMessage `db:"mentions" json:"mentions,omitempty"`
	Attachments      json.RawMessage `db:"attachments" json:"attachments,omitempty"`
	CreatedAt        time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at" json:"updated_at"`
}

func (d *Draft) TableName() string {
	return "messages.drafts"
}

func (d *Draft) PrimaryKey() interface{} {
	return d.ID
}

type Bookmark struct {
	ID             string         `db:"id" json:"id" pk:"true"`
	UserID         string         `db:"user_id" json:"user_id"`
	MessageID      string         `db:"message_id" json:"message_id"`
	CollectionName *string        `db:"collection_name" json:"collection_name,omitempty"`
	Notes          *string        `db:"notes" json:"notes,omitempty"`
	Tags           pq.StringArray `db:"tags" json:"tags,omitempty"`
	BookmarkedAt   time.Time      `db:"bookmarked_at" json:"bookmarked_at"`
}

func (b *Bookmark) TableName() string {
	return "messages.bookmarks"
}

func (b *Bookmark) PrimaryKey() interface{} {
	return b.ID
}

type PinnedMessage struct {
	ID             string    `db:"id" json:"id" pk:"true"`
	ConversationID string    `db:"conversation_id" json:"conversation_id"`
	MessageID      string    `db:"message_id" json:"message_id"`
	PinnedByUserID string    `db:"pinned_by_user_id" json:"pinned_by_user_id"`
	PinOrder       int       `db:"pin_order" json:"pin_order"`
	PinnedAt       time.Time `db:"pinned_at" json:"pinned_at"`
}

func (p *PinnedMessage) TableName() string {
	return "messages.pinned_messages"
}

func (p *PinnedMessage) PrimaryKey() interface{} {
	return p.ID
}

type ConversationInvite struct {
	ID                 string     `db:"id" json:"id" pk:"true"`
	ConversationID     string     `db:"conversation_id" json:"conversation_id"`
	InviterUserID      string     `db:"inviter_user_id" json:"inviter_user_id"`
	InviteeUserID      *string    `db:"invitee_user_id" json:"invitee_user_id,omitempty"`
	InviteePhoneNumber *string    `db:"invitee_phone_number" json:"invitee_phone_number,omitempty"`
	InviteeEmail       *string    `db:"invitee_email" json:"invitee_email,omitempty"`
	InviteCode         *string    `db:"invite_code" json:"invite_code,omitempty"`
	Status             string     `db:"status" json:"status"`
	MaxUses            int        `db:"max_uses" json:"max_uses"`
	UseCount           int        `db:"use_count" json:"use_count"`
	ExpiresAt          *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	AcceptedAt         *time.Time `db:"accepted_at" json:"accepted_at,omitempty"`
	RevokedAt          *time.Time `db:"revoked_at" json:"revoked_at,omitempty"`
	CreatedAt          time.Time  `db:"created_at" json:"created_at"`
}

func (c *ConversationInvite) TableName() string {
	return "messages.conversation_invites"
}

func (c *ConversationInvite) PrimaryKey() interface{} {
	return c.ID
}

type SearchIndex struct {
	ID             string    `db:"id" json:"id" pk:"true"`
	MessageID      string    `db:"message_id" json:"message_id"`
	ConversationID string    `db:"conversation_id" json:"conversation_id"`
	UserID         string    `db:"user_id" json:"user_id"`
	ContentTSV     *string   `db:"content_tsvector" json:"content_tsvector,omitempty"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

func (s *SearchIndex) TableName() string {
	return "messages.search_index"
}

func (s *SearchIndex) PrimaryKey() interface{} {
	return s.ID
}

type Call struct {
	ID                   string          `db:"id" json:"id" pk:"true"`
	ConversationID       string          `db:"conversation_id" json:"conversation_id"`
	CallType             string          `db:"call_type" json:"call_type"`
	InitiatorUserID      string          `db:"initiator_user_id" json:"initiator_user_id"`
	Status               string          `db:"status" json:"status"`
	StartedAt            *time.Time      `db:"started_at" json:"started_at,omitempty"`
	EndedAt              *time.Time      `db:"ended_at" json:"ended_at,omitempty"`
	DurationSeconds      *int            `db:"duration_seconds" json:"duration_seconds,omitempty"`
	VideoQuality         *string         `db:"video_quality" json:"video_quality,omitempty"`
	AudioQuality         *string         `db:"audio_quality" json:"audio_quality,omitempty"`
	ConnectionQuality    *string         `db:"connection_quality" json:"connection_quality,omitempty"`
	PacketLossPercentage *float64        `db:"packet_loss_percentage" json:"packet_loss_percentage,omitempty"`
	MediaServerID        *string         `db:"media_server_id" json:"media_server_id,omitempty"`
	RoomID               *string         `db:"room_id" json:"room_id,omitempty"`
	EndReason            *string         `db:"end_reason" json:"end_reason,omitempty"`
	CreatedAt            time.Time       `db:"created_at" json:"created_at"`
	Metadata             json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

func (c *Call) TableName() string {
	return "messages.calls"
}

func (c *Call) PrimaryKey() interface{} {
	return c.ID
}

type CallParticipant struct {
	ID              string     `db:"id" json:"id" pk:"true"`
	CallID          string     `db:"call_id" json:"call_id"`
	UserID          string     `db:"user_id" json:"user_id"`
	Status          string     `db:"status" json:"status"`
	JoinedAt        *time.Time `db:"joined_at" json:"joined_at,omitempty"`
	LeftAt          *time.Time `db:"left_at" json:"left_at,omitempty"`
	DurationSeconds *int       `db:"duration_seconds" json:"duration_seconds,omitempty"`
	IsVideoEnabled  bool       `db:"is_video_enabled" json:"is_video_enabled"`
	IsAudioEnabled  bool       `db:"is_audio_enabled" json:"is_audio_enabled"`
	IsScreenSharing bool       `db:"is_screen_sharing" json:"is_screen_sharing"`
	RejectionReason *string    `db:"rejection_reason" json:"rejection_reason,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
}

func (c *CallParticipant) TableName() string {
	return "messages.call_participants"
}

func (c *CallParticipant) PrimaryKey() interface{} {
	return c.ID
}

type ConversationSettings struct {
	ID                           string    `db:"id" json:"id" pk:"true"`
	ConversationID               string    `db:"conversation_id" json:"conversation_id"`
	DisappearingMessagesEnabled  bool      `db:"disappearing_messages_enabled" json:"disappearing_messages_enabled"`
	DisappearingMessagesDuration *int      `db:"disappearing_messages_duration" json:"disappearing_messages_duration,omitempty"`
	MessageHistoryEnabled        bool      `db:"message_history_enabled" json:"message_history_enabled"`
	ScreenshotNotification       bool      `db:"screenshot_notification" json:"screenshot_notification"`
	ReadReceiptsEnabled          bool      `db:"read_receipts_enabled" json:"read_receipts_enabled"`
	TypingIndicatorsEnabled      bool      `db:"typing_indicators_enabled" json:"typing_indicators_enabled"`
	LinkPreviewsEnabled          bool      `db:"link_previews_enabled" json:"link_previews_enabled"`
	AutoDownloadMedia            bool      `db:"auto_download_media" json:"auto_download_media"`
	MessageRequestEnabled        bool      `db:"message_request_enabled" json:"message_request_enabled"`
	CreatedAt                    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt                    time.Time `db:"updated_at" json:"updated_at"`
}

func (c *ConversationSettings) TableName() string {
	return "messages.conversation_settings"
}

func (c *ConversationSettings) PrimaryKey() interface{} {
	return c.ID
}
