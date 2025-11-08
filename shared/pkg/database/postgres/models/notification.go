package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

type Notification struct {
	ID     string `db:"id" json:"id" pk:"true"`
	UserID string `db:"user_id" json:"user_id"`

	// Notification details
	NotificationType string  `db:"notification_type" json:"notification_type"`
	NotificationCategory *string `db:"notification_category" json:"notification_category,omitempty"`
	Title            string  `db:"title" json:"title"`
	Body             string  `db:"body" json:"body"`
	Summary          *string `db:"summary" json:"summary,omitempty"`

	// Content
	IconURL    *string `db:"icon_url" json:"icon_url,omitempty"`
	ImageURL   *string `db:"image_url" json:"image_url,omitempty"`
	Sound      string  `db:"sound" json:"sound"`
	BadgeCount *int    `db:"badge_count" json:"badge_count,omitempty"`

	// Related entities
	RelatedUserID         *string `db:"related_user_id" json:"related_user_id,omitempty"`
	RelatedMessageID      *string `db:"related_message_id" json:"related_message_id,omitempty"`
	RelatedConversationID *string `db:"related_conversation_id" json:"related_conversation_id,omitempty"`
	RelatedCallID         *string `db:"related_call_id" json:"related_call_id,omitempty"`

	// Action data
	ActionURL  *string         `db:"action_url" json:"action_url,omitempty"`
	ActionType *string         `db:"action_type" json:"action_type,omitempty"`
	ActionData json.RawMessage `db:"action_data" json:"action_data,omitempty"`

	// Status
	IsRead bool       `db:"is_read" json:"is_read"`
	IsSeen bool       `db:"is_seen" json:"is_seen"`
	ReadAt *time.Time `db:"read_at" json:"read_at,omitempty"`
	SeenAt *time.Time `db:"seen_at" json:"seen_at,omitempty"`

	// Delivery
	DeliveryStatus NotificationDeliveryStatus `db:"delivery_status" json:"delivery_status"`
	SentAt         *time.Time                 `db:"sent_at" json:"sent_at,omitempty"`
	DeliveredAt    *time.Time                 `db:"delivered_at" json:"delivered_at,omitempty"`
	FailedReason   *string                    `db:"failed_reason" json:"failed_reason,omitempty"`
	RetryCount     int                        `db:"retry_count" json:"retry_count"`

	// Priority & Scheduling
	Priority     NotificationPriority `db:"priority" json:"priority"`
	ScheduledFor *time.Time           `db:"scheduled_for" json:"scheduled_for,omitempty"`
	ExpiresAt    *time.Time           `db:"expires_at" json:"expires_at,omitempty"`

	// Grouping
	GroupKey      *string `db:"group_key" json:"group_key,omitempty"`
	GroupCount    int     `db:"group_count" json:"group_count"`
	IsGroupSummary bool   `db:"is_group_summary" json:"is_group_summary"`

	// Device targeting
	DeviceID *string `db:"device_id" json:"device_id,omitempty"`
	Platform *string `db:"platform" json:"platform,omitempty"`

	CreatedAt time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt time.Time        `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time       `db:"deleted_at" json:"deleted_at,omitempty"`
	Metadata  json.RawMessage  `db:"metadata" json:"metadata,omitempty"`
}

func (n *Notification) TableName() string {
	return "notifications.notifications"
}

func (n *Notification) PrimaryKey() interface{} {
	return n.ID
}

type PushDeliveryLog struct {
	ID             string  `db:"id" json:"id" pk:"true"`
	NotificationID string  `db:"notification_id" json:"notification_id"`
	UserID         string  `db:"user_id" json:"user_id"`
	DeviceID       *string `db:"device_id" json:"device_id,omitempty"`

	// Token info
	PushToken    string `db:"push_token" json:"push_token"`
	PushProvider string `db:"push_provider" json:"push_provider"`

	// Delivery details
	Status      PushDeliveryStatus `db:"status" json:"status"`
	SentAt      *time.Time         `db:"sent_at" json:"sent_at,omitempty"`
	DeliveredAt *time.Time         `db:"delivered_at" json:"delivered_at,omitempty"`
	OpenedAt    *time.Time         `db:"opened_at" json:"opened_at,omitempty"`
	DismissedAt *time.Time         `db:"dismissed_at" json:"dismissed_at,omitempty"`

	// Response from provider
	ProviderMessageID *string         `db:"provider_message_id" json:"provider_message_id,omitempty"`
	ProviderResponse  json.RawMessage `db:"provider_response" json:"provider_response,omitempty"`
	ErrorCode         *string         `db:"error_code" json:"error_code,omitempty"`
	ErrorMessage      *string         `db:"error_message" json:"error_message,omitempty"`

	// Metrics
	TimeToDeliverMS *int `db:"time_to_deliver_ms" json:"time_to_deliver_ms,omitempty"`
	TimeToOpenMS    *int `db:"time_to_open_ms" json:"time_to_open_ms,omitempty"`

	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	Metadata  json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

func (p *PushDeliveryLog) TableName() string {
	return "notifications.push_delivery_log"
}

func (p *PushDeliveryLog) PrimaryKey() interface{} {
	return p.ID
}

type EmailNotification struct {
	ID             string  `db:"id" json:"id" pk:"true"`
	UserID         string  `db:"user_id" json:"user_id"`
	NotificationID *string `db:"notification_id" json:"notification_id,omitempty"`

	// Email details
	EmailTo   string  `db:"email_to" json:"email_to"`
	EmailFrom string  `db:"email_from" json:"email_from"`
	ReplyTo   *string `db:"reply_to" json:"reply_to,omitempty"`
	Subject   string  `db:"subject" json:"subject"`
	BodyText  string  `db:"body_text" json:"body_text"`
	BodyHTML  *string `db:"body_html" json:"body_html,omitempty"`

	// Template
	TemplateName *string         `db:"template_name" json:"template_name,omitempty"`
	TemplateData json.RawMessage `db:"template_data" json:"template_data,omitempty"`

	// Status
	Status       EmailStatus `db:"status" json:"status"`
	SentAt       *time.Time  `db:"sent_at" json:"sent_at,omitempty"`
	DeliveredAt  *time.Time  `db:"delivered_at" json:"delivered_at,omitempty"`
	OpenedAt     *time.Time  `db:"opened_at" json:"opened_at,omitempty"`
	ClickedAt    *time.Time  `db:"clicked_at" json:"clicked_at,omitempty"`
	BouncedAt    *time.Time  `db:"bounced_at" json:"bounced_at,omitempty"`
	BounceReason *string     `db:"bounce_reason" json:"bounce_reason,omitempty"`

	// Provider details
	Provider         string          `db:"provider" json:"provider"`
	ProviderMessageID *string        `db:"provider_message_id" json:"provider_message_id,omitempty"`
	ProviderResponse json.RawMessage `db:"provider_response" json:"provider_response,omitempty"`

	// Tracking
	OpenCount  int `db:"open_count" json:"open_count"`
	ClickCount int `db:"click_count" json:"click_count"`

	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	Metadata  json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

func (e *EmailNotification) TableName() string {
	return "notifications.email_notifications"
}

func (e *EmailNotification) PrimaryKey() interface{} {
	return e.ID
}

type SMSNotification struct {
	ID             string  `db:"id" json:"id" pk:"true"`
	UserID         string  `db:"user_id" json:"user_id"`
	NotificationID *string `db:"notification_id" json:"notification_id,omitempty"`

	// SMS details
	PhoneNumber string  `db:"phone_number" json:"phone_number"`
	CountryCode *string `db:"country_code" json:"country_code,omitempty"`
	Message     string  `db:"message" json:"message"`

	// Status
	Status        SMSStatus  `db:"status" json:"status"`
	SentAt        *time.Time `db:"sent_at" json:"sent_at,omitempty"`
	DeliveredAt   *time.Time `db:"delivered_at" json:"delivered_at,omitempty"`
	FailedAt      *time.Time `db:"failed_at" json:"failed_at,omitempty"`
	FailureReason *string    `db:"failure_reason" json:"failure_reason,omitempty"`

	// Provider details
	Provider          string          `db:"provider" json:"provider"`
	ProviderMessageID *string         `db:"provider_message_id" json:"provider_message_id,omitempty"`
	ProviderResponse  json.RawMessage `db:"provider_response" json:"provider_response,omitempty"`

	// Cost tracking
	SegmentCount int      `db:"segment_count" json:"segment_count"`
	CostPerSMS   *float64 `db:"cost_per_sms" json:"cost_per_sms,omitempty"`
	TotalCost    *float64 `db:"total_cost" json:"total_cost,omitempty"`

	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	Metadata  json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

func (s *SMSNotification) TableName() string {
	return "notifications.sms_notifications"
}

func (s *SMSNotification) PrimaryKey() interface{} {
	return s.ID
}

type UserPreference struct {
	ID     string `db:"id" json:"id" pk:"true"`
	UserID string `db:"user_id" json:"user_id"`

	// Global settings
	PushEnabled   bool `db:"push_enabled" json:"push_enabled"`
	EmailEnabled  bool `db:"email_enabled" json:"email_enabled"`
	SMSEnabled    bool `db:"sms_enabled" json:"sms_enabled"`
	InAppEnabled  bool `db:"in_app_enabled" json:"in_app_enabled"`

	// Message notifications
	MessagePush  bool `db:"message_push" json:"message_push"`
	MessageEmail bool `db:"message_email" json:"message_email"`
	MessageSMS   bool `db:"message_sms" json:"message_sms"`

	// Mention notifications
	MentionPush  bool `db:"mention_push" json:"mention_push"`
	MentionEmail bool `db:"mention_email" json:"mention_email"`
	MentionSMS   bool `db:"mention_sms" json:"mention_sms"`

	// Reaction notifications
	ReactionPush  bool `db:"reaction_push" json:"reaction_push"`
	ReactionEmail bool `db:"reaction_email" json:"reaction_email"`
	ReactionSMS   bool `db:"reaction_sms" json:"reaction_sms"`

	// Call notifications
	CallPush        bool `db:"call_push" json:"call_push"`
	CallEmail       bool `db:"call_email" json:"call_email"`
	CallSMS         bool `db:"call_sms" json:"call_sms"`
	MissedCallPush  bool `db:"missed_call_push" json:"missed_call_push"`

	// Social notifications
	FriendRequestPush  bool `db:"friend_request_push" json:"friend_request_push"`
	FriendRequestEmail bool `db:"friend_request_email" json:"friend_request_email"`
	FriendAcceptPush   bool `db:"friend_accept_push" json:"friend_accept_push"`

	// Group notifications
	GroupInvitePush   bool `db:"group_invite_push" json:"group_invite_push"`
	GroupInviteEmail  bool `db:"group_invite_email" json:"group_invite_email"`
	GroupMessagePush  bool `db:"group_message_push" json:"group_message_push"`
	GroupMentionPush  bool `db:"group_mention_push" json:"group_mention_push"`

	// System notifications
	SecurityAlertsPush  bool `db:"security_alerts_push" json:"security_alerts_push"`
	SecurityAlertsEmail bool `db:"security_alerts_email" json:"security_alerts_email"`
	SecurityAlertsSMS   bool `db:"security_alerts_sms" json:"security_alerts_sms"`
	AccountUpdatesEmail bool `db:"account_updates_email" json:"account_updates_email"`

	// Marketing
	MarketingPush      bool `db:"marketing_push" json:"marketing_push"`
	MarketingEmail     bool `db:"marketing_email" json:"marketing_email"`
	PromotionalEmail   bool `db:"promotional_email" json:"promotional_email"`

	// Quiet hours
	QuietHoursEnabled  bool           `db:"quiet_hours_enabled" json:"quiet_hours_enabled"`
	QuietHoursStart    *time.Time     `db:"quiet_hours_start" json:"quiet_hours_start,omitempty"`
	QuietHoursEnd      *time.Time     `db:"quiet_hours_end" json:"quiet_hours_end,omitempty"`
	QuietHoursTimezone *string        `db:"quiet_hours_timezone" json:"quiet_hours_timezone,omitempty"`
	QuietHoursDays     pq.Int64Array  `db:"quiet_hours_days" json:"quiet_hours_days,omitempty"`

	// Notification bundling
	BundleNotifications  bool `db:"bundle_notifications" json:"bundle_notifications"`
	BundleIntervalMinutes int `db:"bundle_interval_minutes" json:"bundle_interval_minutes"`

	// Sound & Vibration
	NotificationSound string `db:"notification_sound" json:"notification_sound"`
	VibrationEnabled  bool   `db:"vibration_enabled" json:"vibration_enabled"`
	LEDNotification   bool   `db:"led_notification" json:"led_notification"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func (u *UserPreference) TableName() string {
	return "notifications.user_preferences"
}

func (u *UserPreference) PrimaryKey() interface{} {
	return u.ID
}

type ConversationChannel struct {
	ID             string `db:"id" json:"id" pk:"true"`
	UserID         string `db:"user_id" json:"user_id"`
	ConversationID string `db:"conversation_id" json:"conversation_id"`

	// Override settings
	NotificationsEnabled *bool   `db:"notifications_enabled" json:"notifications_enabled,omitempty"`
	PushEnabled          *bool   `db:"push_enabled" json:"push_enabled,omitempty"`
	EmailEnabled         *bool   `db:"email_enabled" json:"email_enabled,omitempty"`
	SoundEnabled         *bool   `db:"sound_enabled" json:"sound_enabled,omitempty"`
	VibrationEnabled     *bool   `db:"vibration_enabled" json:"vibration_enabled,omitempty"`

	// Custom sound
	CustomSound *string `db:"custom_sound" json:"custom_sound,omitempty"`

	// Mute settings
	IsMuted     bool       `db:"is_muted" json:"is_muted"`
	MutedUntil  *time.Time `db:"muted_until" json:"muted_until,omitempty"`
	MuteReason  *string    `db:"mute_reason" json:"mute_reason,omitempty"`

	// Priority
	PriorityLevel string `db:"priority_level" json:"priority_level"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func (c *ConversationChannel) TableName() string {
	return "notifications.conversation_channels"
}

func (c *ConversationChannel) PrimaryKey() interface{} {
	return c.ID
}

type Template struct {
	ID              string `db:"id" json:"id" pk:"true"`
	TemplateName    string `db:"template_name" json:"template_name"`
	TemplateType    string `db:"template_type" json:"template_type"`

	// Content
	TitleTemplate *string `db:"title_template" json:"title_template,omitempty"`
	BodyTemplate  string  `db:"body_template" json:"body_template"`
	HTMLTemplate  *string `db:"html_template" json:"html_template,omitempty"`

	// Variables
	RequiredVariables pq.StringArray `db:"required_variables" json:"required_variables,omitempty"`
	OptionalVariables pq.StringArray `db:"optional_variables" json:"optional_variables,omitempty"`

	// Localization
	LanguageCode string `db:"language_code" json:"language_code"`

	// Status
	IsActive bool `db:"is_active" json:"is_active"`
	Version  int  `db:"version" json:"version"`

	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
	CreatedByUserID *string   `db:"created_by_user_id" json:"created_by_user_id,omitempty"`
}

func (t *Template) TableName() string {
	return "notifications.templates"
}

func (t *Template) PrimaryKey() interface{} {
	return t.ID
}

type NotificationAction struct {
	ID             string          `db:"id" json:"id" pk:"true"`
	NotificationID string          `db:"notification_id" json:"notification_id"`
	ActionID       string          `db:"action_id" json:"action_id"`
	ActionLabel    string          `db:"action_label" json:"action_label"`
	ActionType     *string         `db:"action_type" json:"action_type,omitempty"`
	ActionURL      *string         `db:"action_url" json:"action_url,omitempty"`
	ActionData     json.RawMessage `db:"action_data" json:"action_data,omitempty"`
	DisplayOrder   int             `db:"display_order" json:"display_order"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
}

func (n *NotificationAction) TableName() string {
	return "notifications.notification_actions"
}

func (n *NotificationAction) PrimaryKey() interface{} {
	return n.ID
}

type ActionResponse struct {
	ID             string          `db:"id" json:"id" pk:"true"`
	NotificationID string          `db:"notification_id" json:"notification_id"`
	ActionID       string          `db:"action_id" json:"action_id"`
	UserID         string          `db:"user_id" json:"user_id"`
	ResponseType   *string         `db:"response_type" json:"response_type,omitempty"`
	ResponseData   json.RawMessage `db:"response_data" json:"response_data,omitempty"`
	DeviceID       *string         `db:"device_id" json:"device_id,omitempty"`
	RespondedAt    time.Time       `db:"responded_at" json:"responded_at"`
}

func (a *ActionResponse) TableName() string {
	return "notifications.action_responses"
}

func (a *ActionResponse) PrimaryKey() interface{} {
	return a.ID
}

type Batch struct {
	ID         string `db:"id" json:"id" pk:"true"`
	BatchName  *string `db:"batch_name" json:"batch_name,omitempty"`
	BatchType  NotificationBatchType `db:"batch_type" json:"batch_type"`

	// Target
	TargetUserIDs pq.StringArray `db:"target_user_ids" json:"target_user_ids,omitempty"`
	TargetSegment *string        `db:"target_segment" json:"target_segment,omitempty"`
	TargetCount   *int           `db:"target_count" json:"target_count,omitempty"`

	// Content
	NotificationData json.RawMessage `db:"notification_data" json:"notification_data"`

	// Status
	Status   NotificationBatchStatus `db:"status" json:"status"`
	Priority NotificationPriority    `db:"priority" json:"priority"`

	// Progress
	SentCount      int `db:"sent_count" json:"sent_count"`
	DeliveredCount int `db:"delivered_count" json:"delivered_count"`
	FailedCount    int `db:"failed_count" json:"failed_count"`
	OpenedCount    int `db:"opened_count" json:"opened_count"`

	// Scheduling
	ScheduledFor *time.Time `db:"scheduled_for" json:"scheduled_for,omitempty"`
	StartedAt    *time.Time `db:"started_at" json:"started_at,omitempty"`
	CompletedAt  *time.Time `db:"completed_at" json:"completed_at,omitempty"`

	// Limits
	RateLimitPerSecond int `db:"rate_limit_per_second" json:"rate_limit_per_second"`

	CreatedAt       time.Time       `db:"created_at" json:"created_at"`
	CreatedByUserID *string         `db:"created_by_user_id" json:"created_by_user_id,omitempty"`
	Metadata        json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

func (b *Batch) TableName() string {
	return "notifications.batches"
}

func (b *Batch) PrimaryKey() interface{} {
	return b.ID
}

type Announcement struct {
	ID              string `db:"id" json:"id" pk:"true"`

	// Content
	Title            string                `db:"title" json:"title"`
	Message          string                `db:"message" json:"message"`
	AnnouncementType *string               `db:"announcement_type" json:"announcement_type,omitempty"`
	Severity         AnnouncementSeverity  `db:"severity" json:"severity"`

	// Display
	IconURL         *string `db:"icon_url" json:"icon_url,omitempty"`
	ImageURL        *string `db:"image_url" json:"image_url,omitempty"`
	BackgroundColor *string `db:"background_color" json:"background_color,omitempty"`

	// Action
	ActionLabel *string `db:"action_label" json:"action_label,omitempty"`
	ActionURL   *string `db:"action_url" json:"action_url,omitempty"`
	ActionType  *string `db:"action_type" json:"action_type,omitempty"`

	// Targeting
	TargetAudience   string         `db:"target_audience" json:"target_audience"`
	TargetUserIDs    pq.StringArray `db:"target_user_ids" json:"target_user_ids,omitempty"`
	MinAppVersion    *string        `db:"min_app_version" json:"min_app_version,omitempty"`
	MaxAppVersion    *string        `db:"max_app_version" json:"max_app_version,omitempty"`
	TargetCountries  pq.StringArray `db:"target_countries" json:"target_countries,omitempty"`

	// Display rules
	DisplayFrequency   AnnouncementDisplayFrequency `db:"display_frequency" json:"display_frequency"`
	MaxDisplayCount    int                          `db:"max_display_count" json:"max_display_count"`
	DisplayPriority    int                          `db:"display_priority" json:"display_priority"`

	// Status
	IsActive      bool `db:"is_active" json:"is_active"`
	IsDismissible bool `db:"is_dismissible" json:"is_dismissible"`

	// Schedule
	StartsAt *time.Time `db:"starts_at" json:"starts_at,omitempty"`
	EndsAt   *time.Time `db:"ends_at" json:"ends_at,omitempty"`

	// Stats
	ViewCount    int `db:"view_count" json:"view_count"`
	ClickCount   int `db:"click_count" json:"click_count"`
	DismissCount int `db:"dismiss_count" json:"dismiss_count"`

	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
	CreatedByUserID *string   `db:"created_by_user_id" json:"created_by_user_id,omitempty"`
}

func (a *Announcement) TableName() string {
	return "notifications.announcements"
}

func (a *Announcement) PrimaryKey() interface{} {
	return a.ID
}

type AnnouncementView struct {
	ID             string     `db:"id" json:"id" pk:"true"`
	AnnouncementID string     `db:"announcement_id" json:"announcement_id"`
	UserID         string     `db:"user_id" json:"user_id"`
	ViewCount      int        `db:"view_count" json:"view_count"`
	FirstViewedAt  time.Time  `db:"first_viewed_at" json:"first_viewed_at"`
	LastViewedAt   time.Time  `db:"last_viewed_at" json:"last_viewed_at"`
	Clicked        bool       `db:"clicked" json:"clicked"`
	ClickedAt      *time.Time `db:"clicked_at" json:"clicked_at,omitempty"`
	Dismissed      bool       `db:"dismissed" json:"dismissed"`
	DismissedAt    *time.Time `db:"dismissed_at" json:"dismissed_at,omitempty"`
}

func (a *AnnouncementView) TableName() string {
	return "notifications.announcement_views"
}

func (a *AnnouncementView) PrimaryKey() interface{} {
	return a.ID
}

type UserStat struct {
	ID     string `db:"id" json:"id" pk:"true"`
	UserID string `db:"user_id" json:"user_id"`

	TotalNotificationsSent      int `db:"total_notifications_sent" json:"total_notifications_sent"`
	TotalNotificationsDelivered int `db:"total_notifications_delivered" json:"total_notifications_delivered"`
	TotalNotificationsOpened    int `db:"total_notifications_opened" json:"total_notifications_opened"`
	TotalNotificationsDismissed int `db:"total_notifications_dismissed" json:"total_notifications_dismissed"`

	PushSent      int `db:"push_sent" json:"push_sent"`
	PushDelivered int `db:"push_delivered" json:"push_delivered"`
	PushOpened    int `db:"push_opened" json:"push_opened"`

	EmailSent      int `db:"email_sent" json:"email_sent"`
	EmailDelivered int `db:"email_delivered" json:"email_delivered"`
	EmailOpened    int `db:"email_opened" json:"email_opened"`
	EmailClicked   int `db:"email_clicked" json:"email_clicked"`

	SMSSent      int `db:"sms_sent" json:"sms_sent"`
	SMSDelivered int `db:"sms_delivered" json:"sms_delivered"`

	LastNotificationAt       *time.Time `db:"last_notification_at" json:"last_notification_at,omitempty"`
	LastOpenedNotificationAt *time.Time `db:"last_opened_notification_at" json:"last_opened_notification_at,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func (u *UserStat) TableName() string {
	return "notifications.user_stats"
}

func (u *UserStat) PrimaryKey() interface{} {
	return u.ID
}

type Subscription struct {
	ID            string     `db:"id" json:"id" pk:"true"`
	UserID        string     `db:"user_id" json:"user_id"`
	TopicName     string     `db:"topic_name" json:"topic_name"`
	IsSubscribed  bool       `db:"is_subscribed" json:"is_subscribed"`
	SubscribedAt  time.Time  `db:"subscribed_at" json:"subscribed_at"`
	UnsubscribedAt *time.Time `db:"unsubscribed_at" json:"unsubscribed_at,omitempty"`
}

func (s *Subscription) TableName() string {
	return "notifications.subscriptions"
}

func (s *Subscription) PrimaryKey() interface{} {
	return s.ID
}