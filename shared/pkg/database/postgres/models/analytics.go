package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

type Event struct {
	ID         string `db:"id" json:"id" pk:"true"`
	UserID     *string `db:"user_id" json:"user_id,omitempty"`
	SessionID  *string `db:"session_id" json:"session_id,omitempty"`

	// Event details
	EventName     string   `db:"event_name" json:"event_name"`
	EventCategory *string  `db:"event_category" json:"event_category,omitempty"`
	EventAction   *string  `db:"event_action" json:"event_action,omitempty"`
	EventLabel    *string  `db:"event_label" json:"event_label,omitempty"`
	EventValue    *float64 `db:"event_value" json:"event_value,omitempty"`

	// Context
	ScreenName     *string `db:"screen_name" json:"screen_name,omitempty"`
	PreviousScreen *string `db:"previous_screen" json:"previous_screen,omitempty"`
	FlowName       *string `db:"flow_name" json:"flow_name,omitempty"`

	// Device & Platform
	DeviceID   *string `db:"device_id" json:"device_id,omitempty"`
	Platform   *string `db:"platform" json:"platform,omitempty"`
	OSName     *string `db:"os_name" json:"os_name,omitempty"`
	OSVersion  *string `db:"os_version" json:"os_version,omitempty"`
	AppVersion *string `db:"app_version" json:"app_version,omitempty"`
	AppBuild   *string `db:"app_build" json:"app_build,omitempty"`

	// Location
	IPAddress *string  `db:"ip_address" json:"ip_address,omitempty"`
	Country   *string  `db:"country" json:"country,omitempty"`
	Region    *string  `db:"region" json:"region,omitempty"`
	City      *string  `db:"city" json:"city,omitempty"`
	Timezone  *string  `db:"timezone" json:"timezone,omitempty"`
	Latitude  *float64 `db:"latitude" json:"latitude,omitempty"`
	Longitude *float64 `db:"longitude" json:"longitude,omitempty"`

	// Network
	ConnectionType *string `db:"connection_type" json:"connection_type,omitempty"`
	Carrier        *string `db:"carrier" json:"carrier,omitempty"`

	// Performance
	EventDurationMS *int `db:"event_duration_ms" json:"event_duration_ms,omitempty"`
	LoadTimeMS      *int `db:"load_time_ms" json:"load_time_ms,omitempty"`
	TTFBMS          *int `db:"ttfb_ms" json:"ttfb_ms,omitempty"`

	// Custom properties
	Properties json.RawMessage `db:"properties" json:"properties,omitempty"`

	// Attribution
	UTMSource   *string `db:"utm_source" json:"utm_source,omitempty"`
	UTMMedium   *string `db:"utm_medium" json:"utm_medium,omitempty"`
	UTMCampaign *string `db:"utm_campaign" json:"utm_campaign,omitempty"`
	UTMTerm     *string `db:"utm_term" json:"utm_term,omitempty"`
	UTMContent  *string `db:"utm_content" json:"utm_content,omitempty"`
	Referrer    *string `db:"referrer" json:"referrer,omitempty"`

	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	EventTimestamp time.Time `db:"event_timestamp" json:"event_timestamp"`
}

func (e *Event) TableName() string {
	return "analytics.events"
}

func (e *Event) PrimaryKey() interface{} {
	return e.ID
}

type UserSession struct {
	ID        string  `db:"id" json:"id" pk:"true"`
	UserID    string  `db:"user_id" json:"user_id"`
	SessionID string  `db:"session_id" json:"session_id"`

	// Session details
	SessionStart          time.Time  `db:"session_start" json:"session_start"`
	SessionEnd            *time.Time `db:"session_end" json:"session_end,omitempty"`
	SessionDurationSeconds *int      `db:"session_duration_seconds" json:"session_duration_seconds,omitempty"`

	// Activity
	PageViews        int `db:"page_views" json:"page_views"`
	ScreenViews      int `db:"screen_views" json:"screen_views"`
	EventCount       int `db:"event_count" json:"event_count"`
	MessagesSent     int `db:"messages_sent" json:"messages_sent"`
	MessagesReceived int `db:"messages_received" json:"messages_received"`

	// Device & Platform
	DeviceID   *string `db:"device_id" json:"device_id,omitempty"`
	DeviceType *string `db:"device_type" json:"device_type,omitempty"`
	Platform   *string `db:"platform" json:"platform,omitempty"`
	AppVersion *string `db:"app_version" json:"app_version,omitempty"`

	// Location
	Country   *string `db:"country" json:"country,omitempty"`
	City      *string `db:"city" json:"city,omitempty"`
	IPAddress *string `db:"ip_address" json:"ip_address,omitempty"`

	// Engagement
	IsEngaged bool `db:"is_engaged" json:"is_engaged"`
	Bounce    bool `db:"bounce" json:"bounce"`

	// Attribution
	TrafficSource *string `db:"traffic_source" json:"traffic_source,omitempty"`
	Campaign      *string `db:"campaign" json:"campaign,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (u *UserSession) TableName() string {
	return "analytics.user_sessions"
}

func (u *UserSession) PrimaryKey() interface{} {
	return u.ID
}

type DailyActiveUser struct {
	ID     string    `db:"id" json:"id" pk:"true"`
	Date   time.Time `db:"date" json:"date"`
	UserID string    `db:"user_id" json:"user_id"`

	// Activity metrics
	SessionsCount     int            `db:"sessions_count" json:"sessions_count"`
	MessagesSent      int            `db:"messages_sent" json:"messages_sent"`
	MessagesReceived  int            `db:"messages_received" json:"messages_received"`
	TimeSpentSeconds  int            `db:"time_spent_seconds" json:"time_spent_seconds"`
	FeaturesUsed      pq.StringArray `db:"features_used" json:"features_used,omitempty"`

	// Platform
	PlatformsUsed pq.StringArray `db:"platforms_used" json:"platforms_used,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (d *DailyActiveUser) TableName() string {
	return "analytics.daily_active_users"
}

func (d *DailyActiveUser) PrimaryKey() interface{} {
	return d.ID
}

type UserCohort struct {
	ID          string    `db:"id" json:"id" pk:"true"`
	UserID      string    `db:"user_id" json:"user_id"`
	CohortDate  time.Time `db:"cohort_date" json:"cohort_date"`
	CohortWeek  *int      `db:"cohort_week" json:"cohort_week,omitempty"`
	CohortMonth *int      `db:"cohort_month" json:"cohort_month,omitempty"`

	// Retention tracking
	Day1Active  bool `db:"day_1_active" json:"day_1_active"`
	Day7Active  bool `db:"day_7_active" json:"day_7_active"`
	Day14Active bool `db:"day_14_active" json:"day_14_active"`
	Day30Active bool `db:"day_30_active" json:"day_30_active"`
	Day60Active bool `db:"day_60_active" json:"day_60_active"`
	Day90Active bool `db:"day_90_active" json:"day_90_active"`

	// Engagement
	MessagesSentTotal int        `db:"messages_sent_total" json:"messages_sent_total"`
	DaysActiveCount   int        `db:"days_active_count" json:"days_active_count"`
	LastActiveDate    *time.Time `db:"last_active_date" json:"last_active_date,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func (u *UserCohort) TableName() string {
	return "analytics.user_cohorts"
}

func (u *UserCohort) PrimaryKey() interface{} {
	return u.ID
}

type Funnel struct {
	ID        string  `db:"id" json:"id" pk:"true"`
	UserID    *string `db:"user_id" json:"user_id,omitempty"`
	SessionID *string `db:"session_id" json:"session_id,omitempty"`

	FunnelName string `db:"funnel_name" json:"funnel_name"`
	StepName   string `db:"step_name" json:"step_name"`
	StepOrder  int    `db:"step_order" json:"step_order"`

	// Timing
	EnteredAt       time.Time  `db:"entered_at" json:"entered_at"`
	CompletedAt     *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	TimeInStepSeconds *int     `db:"time_in_step_seconds" json:"time_in_step_seconds,omitempty"`

	// Status
	IsCompleted    bool    `db:"is_completed" json:"is_completed"`
	DroppedOff     bool    `db:"dropped_off" json:"dropped_off"`
	DropOffReason  *string `db:"drop_off_reason" json:"drop_off_reason,omitempty"`

	// Context
	Properties json.RawMessage `db:"properties" json:"properties,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (f *Funnel) TableName() string {
	return "analytics.funnels"
}

func (f *Funnel) PrimaryKey() interface{} {
	return f.ID
}

type FeatureUsage struct {
	ID     string `db:"id" json:"id" pk:"true"`
	UserID string `db:"user_id" json:"user_id"`

	FeatureName     string  `db:"feature_name" json:"feature_name"`
	FeatureCategory *string `db:"feature_category" json:"feature_category,omitempty"`

	// Usage stats
	UsageCount           int       `db:"usage_count" json:"usage_count"`
	FirstUsedAt          time.Time `db:"first_used_at" json:"first_used_at"`
	LastUsedAt           time.Time `db:"last_used_at" json:"last_used_at"`
	TotalDurationSeconds int       `db:"total_duration_seconds" json:"total_duration_seconds"`

	// Value
	EngagementScore *float64 `db:"engagement_score" json:"engagement_score,omitempty"`

	Date time.Time `db:"date" json:"date"`
}

func (f *FeatureUsage) TableName() string {
	return "analytics.feature_usage"
}

func (f *FeatureUsage) PrimaryKey() interface{} {
	return f.ID
}

type ABTest struct {
	ID              string  `db:"id" json:"id" pk:"true"`
	TestName        string  `db:"test_name" json:"test_name"`
	TestDescription *string `db:"test_description" json:"test_description,omitempty"`

	// Configuration
	Variants json.RawMessage `db:"variants" json:"variants"`

	// Targeting
	TargetPercentage    int            `db:"target_percentage" json:"target_percentage"`
	TargetPlatforms     pq.StringArray `db:"target_platforms" json:"target_platforms,omitempty"`
	TargetCountries     pq.StringArray `db:"target_countries" json:"target_countries,omitempty"`
	TargetUserSegments  pq.StringArray `db:"target_user_segments" json:"target_user_segments,omitempty"`

	// Status
	Status ABTestStatus `db:"status" json:"status"`

	// Schedule
	StartsAt *time.Time `db:"starts_at" json:"starts_at,omitempty"`
	EndsAt   *time.Time `db:"ends_at" json:"ends_at,omitempty"`

	// Results
	SampleSize       int      `db:"sample_size" json:"sample_size"`
	ConfidenceLevel  *float64 `db:"confidence_level" json:"confidence_level,omitempty"`
	WinningVariant   *string  `db:"winning_variant" json:"winning_variant,omitempty"`

	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
	CreatedByUserID *string   `db:"created_by_user_id" json:"created_by_user_id,omitempty"`
}

func (a *ABTest) TableName() string {
	return "analytics.ab_tests"
}

func (a *ABTest) PrimaryKey() interface{} {
	return a.ID
}

type ABTestAssignment struct {
	ID     string `db:"id" json:"id" pk:"true"`
	TestID string `db:"test_id" json:"test_id"`
	UserID string `db:"user_id" json:"user_id"`

	VariantName   string          `db:"variant_name" json:"variant_name"`
	VariantConfig json.RawMessage `db:"variant_config" json:"variant_config,omitempty"`

	// Metrics
	Converted       bool       `db:"converted" json:"converted"`
	ConversionValue *float64   `db:"conversion_value" json:"conversion_value,omitempty"`
	ConvertedAt     *time.Time `db:"converted_at" json:"converted_at,omitempty"`

	AssignedAt time.Time `db:"assigned_at" json:"assigned_at"`
}

func (a *ABTestAssignment) TableName() string {
	return "analytics.ab_test_assignments"
}

func (a *ABTestAssignment) PrimaryKey() interface{} {
	return a.ID
}

type PerformanceMetric struct {
	ID string `db:"id" json:"id" pk:"true"`

	// Metric details
	MetricType  string  `db:"metric_type" json:"metric_type"`
	MetricName  string  `db:"metric_name" json:"metric_name"`
	MetricValue float64 `db:"metric_value" json:"metric_value"`
	MetricUnit  *string `db:"metric_unit" json:"metric_unit,omitempty"`

	// Context
	ServiceName *string `db:"service_name" json:"service_name,omitempty"`
	Endpoint    *string `db:"endpoint" json:"endpoint,omitempty"`
	Method      *string `db:"method" json:"method,omitempty"`
	StatusCode  *int    `db:"status_code" json:"status_code,omitempty"`

	// Performance
	DurationMS    *int     `db:"duration_ms" json:"duration_ms,omitempty"`
	MemoryUsedMB  *int     `db:"memory_used_mb" json:"memory_used_mb,omitempty"`
	CPUPercentage *float64 `db:"cpu_percentage" json:"cpu_percentage,omitempty"`

	// Request details
	UserID    *string `db:"user_id" json:"user_id,omitempty"`
	SessionID *string `db:"session_id" json:"session_id,omitempty"`
	IPAddress *string `db:"ip_address" json:"ip_address,omitempty"`

	// Error tracking
	IsError      bool    `db:"is_error" json:"is_error"`
	ErrorMessage *string `db:"error_message" json:"error_message,omitempty"`
	ErrorStack   *string `db:"error_stack" json:"error_stack,omitempty"`

	Timestamp time.Time `db:"timestamp" json:"timestamp"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (p *PerformanceMetric) TableName() string {
	return "analytics.performance_metrics"
}

func (p *PerformanceMetric) PrimaryKey() interface{} {
	return p.ID
}

type ErrorLog struct {
	ID string `db:"id" json:"id" pk:"true"`

	// Error details
	ErrorType    string  `db:"error_type" json:"error_type"`
	ErrorMessage string  `db:"error_message" json:"error_message"`
	ErrorCode    *string `db:"error_code" json:"error_code,omitempty"`
	ErrorStack   *string `db:"error_stack" json:"error_stack,omitempty"`

	// Severity
	Severity ErrorSeverity `db:"severity" json:"severity"`

	// Context
	ServiceName  *string `db:"service_name" json:"service_name,omitempty"`
	FunctionName *string `db:"function_name" json:"function_name,omitempty"`
	FilePath     *string `db:"file_path" json:"file_path,omitempty"`
	LineNumber   *int    `db:"line_number" json:"line_number,omitempty"`

	// User context
	UserID    *string `db:"user_id" json:"user_id,omitempty"`
	SessionID *string `db:"session_id" json:"session_id,omitempty"`
	DeviceID  *string `db:"device_id" json:"device_id,omitempty"`

	// Request context
	HTTPMethod   *string         `db:"http_method" json:"http_method,omitempty"`
	Endpoint     *string         `db:"endpoint" json:"endpoint,omitempty"`
	RequestID    *string         `db:"request_id" json:"request_id,omitempty"`
	RequestBody  json.RawMessage `db:"request_body" json:"request_body,omitempty"`
	ResponseBody json.RawMessage `db:"response_body" json:"response_body,omitempty"`

	// Environment
	Environment *string `db:"environment" json:"environment,omitempty"`
	AppVersion  *string `db:"app_version" json:"app_version,omitempty"`
	Platform    *string `db:"platform" json:"platform,omitempty"`

	// Frequency
	Occurrences      int       `db:"occurrences" json:"occurrences"`
	FirstOccurredAt  time.Time `db:"first_occurred_at" json:"first_occurred_at"`
	LastOccurredAt   time.Time `db:"last_occurred_at" json:"last_occurred_at"`

	// Resolution
	IsResolved        bool       `db:"is_resolved" json:"is_resolved"`
	ResolvedAt        *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`
	ResolvedByUserID  *string    `db:"resolved_by_user_id" json:"resolved_by_user_id,omitempty"`
	ResolutionNotes   *string    `db:"resolution_notes" json:"resolution_notes,omitempty"`

	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	Metadata  json.RawMessage `db:"metadata" json:"metadata,omitempty"`
}

func (e *ErrorLog) TableName() string {
	return "analytics.error_logs"
}

func (e *ErrorLog) PrimaryKey() interface{} {
	return e.ID
}

type RevenueEvent struct {
	ID     string `db:"id" json:"id" pk:"true"`
	UserID string `db:"user_id" json:"user_id"`

	// Transaction details
	TransactionType  string  `db:"transaction_type" json:"transaction_type"`
	ProductID        *string `db:"product_id" json:"product_id,omitempty"`
	ProductName      *string `db:"product_name" json:"product_name,omitempty"`
	ProductCategory  *string `db:"product_category" json:"product_category,omitempty"`

	// Amount
	Amount    float64  `db:"amount" json:"amount"`
	Currency  string   `db:"currency" json:"currency"`
	AmountUSD *float64 `db:"amount_usd" json:"amount_usd,omitempty"`

	// Payment
	PaymentMethod   *string `db:"payment_method" json:"payment_method,omitempty"`
	PaymentProvider *string `db:"payment_provider" json:"payment_provider,omitempty"`
	TransactionID   *string `db:"transaction_id" json:"transaction_id,omitempty"`

	// Status
	Status        RevenueEventStatus `db:"status" json:"status"`
	RefundedAt    *time.Time         `db:"refunded_at" json:"refunded_at,omitempty"`
	RefundAmount  *float64           `db:"refund_amount" json:"refund_amount,omitempty"`
	RefundReason  *string            `db:"refund_reason" json:"refund_reason,omitempty"`

	// Attribution
	CampaignSource *string `db:"campaign_source" json:"campaign_source,omitempty"`

	// Subscription specific
	IsSubscription     bool    `db:"is_subscription" json:"is_subscription"`
	SubscriptionPeriod *string `db:"subscription_period" json:"subscription_period,omitempty"`
	IsTrial            bool    `db:"is_trial" json:"is_trial"`
	IsRenewal          bool    `db:"is_renewal" json:"is_renewal"`

	TransactionDate time.Time `db:"transaction_date" json:"transaction_date"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

func (r *RevenueEvent) TableName() string {
	return "analytics.revenue_events"
}

func (r *RevenueEvent) PrimaryKey() interface{} {
	return r.ID
}

type UserLTV struct {
	ID     string `db:"id" json:"id" pk:"true"`
	UserID string `db:"user_id" json:"user_id"`

	// Revenue
	TotalRevenue            float64 `db:"total_revenue" json:"total_revenue"`
	TotalTransactions       int     `db:"total_transactions" json:"total_transactions"`
	AverageTransactionValue float64 `db:"average_transaction_value" json:"average_transaction_value"`

	// Engagement
	DaysActive            int `db:"days_active" json:"days_active"`
	MessagesSentTotal     int `db:"messages_sent_total" json:"messages_sent_total"`
	MessagesReceivedTotal int `db:"messages_received_total" json:"messages_received_total"`

	// Predicted LTV
	PredictedLTV30d  *float64 `db:"predicted_ltv_30d" json:"predicted_ltv_30d,omitempty"`
	PredictedLTV90d  *float64 `db:"predicted_ltv_90d" json:"predicted_ltv_90d,omitempty"`
	PredictedLTV365d *float64 `db:"predicted_ltv_365d" json:"predicted_ltv_365d,omitempty"`

	// Segments
	UserSegment    *string  `db:"user_segment" json:"user_segment,omitempty"`
	ChurnRiskScore *float64 `db:"churn_risk_score" json:"churn_risk_score,omitempty"`

	LastCalculatedAt time.Time `db:"last_calculated_at" json:"last_calculated_at"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}

func (u *UserLTV) TableName() string {
	return "analytics.user_ltv"
}

func (u *UserLTV) PrimaryKey() interface{} {
	return u.ID
}

type DailyMetric struct {
	ID   string    `db:"id" json:"id" pk:"true"`
	Date time.Time `db:"date" json:"date"`

	// User metrics
	DAU          int `db:"dau" json:"dau"`
	NewUsers     int `db:"new_users" json:"new_users"`
	ChurnedUsers int `db:"churned_users" json:"churned_users"`

	// Engagement
	TotalMessagesSent        int64   `db:"total_messages_sent" json:"total_messages_sent"`
	TotalSessions            int64   `db:"total_sessions" json:"total_sessions"`
	AvgSessionDurationSeconds int     `db:"avg_session_duration_seconds" json:"avg_session_duration_seconds"`
	AvgMessagesPerUser       float64 `db:"avg_messages_per_user" json:"avg_messages_per_user"`

	// Content
	ImagesUploaded     int `db:"images_uploaded" json:"images_uploaded"`
	VideosUploaded     int `db:"videos_uploaded" json:"videos_uploaded"`
	VoiceMessagesSent  int `db:"voice_messages_sent" json:"voice_messages_sent"`

	// Social
	NewConversations int `db:"new_conversations" json:"new_conversations"`
	NewGroupsCreated int `db:"new_groups_created" json:"new_groups_created"`
	NewFriendships   int `db:"new_friendships" json:"new_friendships"`

	// Calls
	VoiceCallsTotal        int   `db:"voice_calls_total" json:"voice_calls_total"`
	VideoCallsTotal        int   `db:"video_calls_total" json:"video_calls_total"`
	TotalCallDurationMinutes int64 `db:"total_call_duration_minutes" json:"total_call_duration_minutes"`

	// Revenue
	RevenueTotal          float64 `db:"revenue_total" json:"revenue_total"`
	NewSubscriptions      int     `db:"new_subscriptions" json:"new_subscriptions"`
	CancelledSubscriptions int     `db:"cancelled_subscriptions" json:"cancelled_subscriptions"`

	// Performance
	AvgAPILatencyMS   *int     `db:"avg_api_latency_ms" json:"avg_api_latency_ms,omitempty"`
	ErrorCount        int      `db:"error_count" json:"error_count"`
	UptimePercentage  float64  `db:"uptime_percentage" json:"uptime_percentage"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (d *DailyMetric) TableName() string {
	return "analytics.daily_metrics"
}

func (d *DailyMetric) PrimaryKey() interface{} {
	return d.ID
}

type UserSegment struct {
	ID     string `db:"id" json:"id" pk:"true"`
	UserID string `db:"user_id" json:"user_id"`

	SegmentName     string  `db:"segment_name" json:"segment_name"`
	SegmentCategory *string `db:"segment_category" json:"segment_category,omitempty"`

	// Criteria met
	CriteriaMet     json.RawMessage `db:"criteria_met" json:"criteria_met,omitempty"`
	ConfidenceScore *float64        `db:"confidence_score" json:"confidence_score,omitempty"`

	AssignedAt time.Time  `db:"assigned_at" json:"assigned_at"`
	ExpiresAt  *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	IsActive   bool       `db:"is_active" json:"is_active"`
}

func (u *UserSegment) TableName() string {
	return "analytics.user_segments"
}

func (u *UserSegment) PrimaryKey() interface{} {
	return u.ID
}

type PageView struct {
	ID        string  `db:"id" json:"id" pk:"true"`
	UserID    *string `db:"user_id" json:"user_id,omitempty"`
	SessionID *string `db:"session_id" json:"session_id,omitempty"`

	// Page details
	PageURL     *string `db:"page_url" json:"page_url,omitempty"`
	PageTitle   *string `db:"page_title" json:"page_title,omitempty"`
	ScreenName  *string `db:"screen_name" json:"screen_name,omitempty"`
	ScreenClass *string `db:"screen_class" json:"screen_class,omitempty"`

	// Referrer
	ReferrerURL    *string `db:"referrer_url" json:"referrer_url,omitempty"`
	ReferrerSource *string `db:"referrer_source" json:"referrer_source,omitempty"`

	// Timing
	ViewDurationSeconds  *int `db:"view_duration_seconds" json:"view_duration_seconds,omitempty"`
	TimeToInteractiveMS *int `db:"time_to_interactive_ms" json:"time_to_interactive_ms,omitempty"`

	// Engagement
	ScrollDepthPercentage *int `db:"scroll_depth_percentage" json:"scroll_depth_percentage,omitempty"`
	ClicksCount           int  `db:"clicks_count" json:"clicks_count"`

	// Device
	DeviceID *string `db:"device_id" json:"device_id,omitempty"`
	Platform *string `db:"platform" json:"platform,omitempty"`

	ViewedAt  time.Time `db:"viewed_at" json:"viewed_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (p *PageView) TableName() string {
	return "analytics.page_views"
}

func (p *PageView) PrimaryKey() interface{} {
	return p.ID
}

type SearchQuery struct {
	ID     string  `db:"id" json:"id" pk:"true"`
	UserID *string `db:"user_id" json:"user_id,omitempty"`

	// Query details
	SearchQuery    string  `db:"search_query" json:"search_query"`
	SearchType     *string `db:"search_type" json:"search_type,omitempty"`
	SearchCategory *string `db:"search_category" json:"search_category,omitempty"`

	// Results
	ResultsCount   int  `db:"results_count" json:"results_count"`
	ResultsClicked int  `db:"results_clicked" json:"results_clicked"`
	ClickedPosition *int `db:"clicked_position" json:"clicked_position,omitempty"`

	// Timing
	SearchDurationMS *int `db:"search_duration_ms" json:"search_duration_ms,omitempty"`

	// Context
	ScreenName *string `db:"screen_name" json:"screen_name,omitempty"`

	SearchedAt time.Time `db:"searched_at" json:"searched_at"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

func (s *SearchQuery) TableName() string {
	return "analytics.search_queries"
}

func (s *SearchQuery) PrimaryKey() interface{} {
	return s.ID
}

type ContentEngagement struct {
	ID     string `db:"id" json:"id" pk:"true"`
	UserID string `db:"user_id" json:"user_id"`

	ContentType string `db:"content_type" json:"content_type"`
	ContentID   string `db:"content_id" json:"content_id"`

	// Engagement metrics
	Viewed         bool       `db:"viewed" json:"viewed"`
	ViewedAt       *time.Time `db:"viewed_at" json:"viewed_at,omitempty"`
	ViewDurationMS *int       `db:"view_duration_ms" json:"view_duration_ms,omitempty"`

	Liked   bool       `db:"liked" json:"liked"`
	LikedAt *time.Time `db:"liked_at" json:"liked_at,omitempty"`

	Shared     bool       `db:"shared" json:"shared"`
	SharedAt   *time.Time `db:"shared_at" json:"shared_at,omitempty"`
	ShareCount int        `db:"share_count" json:"share_count"`

	Saved   bool       `db:"saved" json:"saved"`
	SavedAt *time.Time `db:"saved_at" json:"saved_at,omitempty"`

	Commented    bool `db:"commented" json:"commented"`
	CommentCount int  `db:"comment_count" json:"comment_count"`

	// Completion (for video/audio)
	CompletionPercentage *int `db:"completion_percentage" json:"completion_percentage,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (c *ContentEngagement) TableName() string {
	return "analytics.content_engagement"
}

func (c *ContentEngagement) PrimaryKey() interface{} {
	return c.ID
}

type PushMetric struct {
	ID   string    `db:"id" json:"id" pk:"true"`
	Date time.Time `db:"date" json:"date"`

	NotificationType *string `db:"notification_type" json:"notification_type,omitempty"`
	Platform         *string `db:"platform" json:"platform,omitempty"`

	// Metrics
	SentCount      int `db:"sent_count" json:"sent_count"`
	DeliveredCount int `db:"delivered_count" json:"delivered_count"`
	OpenedCount    int `db:"opened_count" json:"opened_count"`
	DismissedCount int `db:"dismissed_count" json:"dismissed_count"`
	FailedCount    int `db:"failed_count" json:"failed_count"`

	// Rates
	DeliveryRate float64 `db:"delivery_rate" json:"delivery_rate"`
	OpenRate     float64 `db:"open_rate" json:"open_rate"`

	// Timing
	AvgDeliveryTimeSeconds *int `db:"avg_delivery_time_seconds" json:"avg_delivery_time_seconds,omitempty"`
	AvgTimeToOpenSeconds   *int `db:"avg_time_to_open_seconds" json:"avg_time_to_open_seconds,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (p *PushMetric) TableName() string {
	return "analytics.push_metrics"
}

func (p *PushMetric) PrimaryKey() interface{} {
	return p.ID
}

type APIUsage struct {
	ID string `db:"id" json:"id" pk:"true"`

	// API details
	Endpoint    string  `db:"endpoint" json:"endpoint"`
	Method      string  `db:"method" json:"method"`
	ServiceName *string `db:"service_name" json:"service_name,omitempty"`

	// Authentication
	UserID   *string `db:"user_id" json:"user_id,omitempty"`
	APIKeyID *string `db:"api_key_id" json:"api_key_id,omitempty"`

	// Request
	RequestSizeBytes *int            `db:"request_size_bytes" json:"request_size_bytes,omitempty"`
	RequestHeaders   json.RawMessage `db:"request_headers" json:"request_headers,omitempty"`
	QueryParams      json.RawMessage `db:"query_params" json:"query_params,omitempty"`

	// Response
	StatusCode        int  `db:"status_code" json:"status_code"`
	ResponseSizeBytes *int `db:"response_size_bytes" json:"response_size_bytes,omitempty"`
	ResponseTimeMS    *int `db:"response_time_ms" json:"response_time_ms,omitempty"`

	// Performance
	DBQueryTimeMS *int `db:"db_query_time_ms" json:"db_query_time_ms,omitempty"`
	CacheHit      bool `db:"cache_hit" json:"cache_hit"`

	// Location
	IPAddress *string `db:"ip_address" json:"ip_address,omitempty"`
	Country   *string `db:"country" json:"country,omitempty"`

	// Error
	IsError      bool    `db:"is_error" json:"is_error"`
	ErrorCode    *string `db:"error_code" json:"error_code,omitempty"`
	ErrorMessage *string `db:"error_message" json:"error_message,omitempty"`

	Timestamp time.Time `db:"timestamp" json:"timestamp"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (a *APIUsage) TableName() string {
	return "analytics.api_usage"
}

func (a *APIUsage) PrimaryKey() interface{} {
	return a.ID
}

type WebhookDelivery struct {
	ID         string `db:"id" json:"id" pk:"true"`

	WebhookURL string `db:"webhook_url" json:"webhook_url"`
	EventType  string `db:"event_type" json:"event_type"`

	// Delivery
	Status           WebhookDeliveryStatus `db:"status" json:"status"`
	HTTPStatusCode   *int                  `db:"http_status_code" json:"http_status_code,omitempty"`
	ResponseTimeMS   *int                  `db:"response_time_ms" json:"response_time_ms,omitempty"`

	// Retry logic
	AttemptNumber int        `db:"attempt_number" json:"attempt_number"`
	MaxAttempts   int        `db:"max_attempts" json:"max_attempts"`
	NextRetryAt   *time.Time `db:"next_retry_at" json:"next_retry_at,omitempty"`

	// Payload
	PayloadSizeBytes *int `db:"payload_size_bytes" json:"payload_size_bytes,omitempty"`

	// Error
	ErrorMessage *string `db:"error_message" json:"error_message,omitempty"`

	SentAt    *time.Time `db:"sent_at" json:"sent_at,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
}

func (w *WebhookDelivery) TableName() string {
	return "analytics.webhook_deliveries"
}

func (w *WebhookDelivery) PrimaryKey() interface{} {
	return w.ID
}

type UserFeedback struct {
	ID     string `db:"id" json:"id" pk:"true"`
	UserID string `db:"user_id" json:"user_id"`

	// Feedback type
	FeedbackType     string  `db:"feedback_type" json:"feedback_type"`
	FeedbackCategory *string `db:"feedback_category" json:"feedback_category,omitempty"`

	// Content
	Rating      *int    `db:"rating" json:"rating,omitempty"`
	Title       *string `db:"title" json:"title,omitempty"`
	Description *string `db:"description" json:"description,omitempty"`

	// Context
	ScreenName *string `db:"screen_name" json:"screen_name,omitempty"`
	AppVersion *string `db:"app_version" json:"app_version,omitempty"`
	Platform   *string `db:"platform" json:"platform,omitempty"`

	// Attachments
	ScreenshotURLs pq.StringArray  `db:"screenshot_urls" json:"screenshot_urls,omitempty"`
	LogData        json.RawMessage `db:"log_data" json:"log_data,omitempty"`

	// Status
	Status   UserFeedbackStatus   `db:"status" json:"status"`
	Priority UserFeedbackPriority `db:"priority" json:"priority"`

	// Response
	ResponseText       *string    `db:"response_text" json:"response_text,omitempty"`
	RespondedAt        *time.Time `db:"responded_at" json:"responded_at,omitempty"`
	RespondedByUserID  *string    `db:"responded_by_user_id" json:"responded_by_user_id,omitempty"`

	// Sentiment
	SentimentScore *float64 `db:"sentiment_score" json:"sentiment_score,omitempty"`
	SentimentLabel *string  `db:"sentiment_label" json:"sentiment_label,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func (u *UserFeedback) TableName() string {
	return "analytics.user_feedback"
}

func (u *UserFeedback) PrimaryKey() interface{} {
	return u.ID
}

type Referral struct {
	ID             string  `db:"id" json:"id" pk:"true"`
	ReferrerUserID string  `db:"referrer_user_id" json:"referrer_user_id"`
	ReferredUserID *string `db:"referred_user_id" json:"referred_user_id,omitempty"`

	// Referral method
	ReferralCode   string  `db:"referral_code" json:"referral_code"`
	ReferralLink   *string `db:"referral_link" json:"referral_link,omitempty"`
	ReferralMethod *string `db:"referral_method" json:"referral_method,omitempty"`

	// Status
	Status ReferralStatus `db:"status" json:"status"`

	// Tracking
	ClickedAt    *time.Time `db:"clicked_at" json:"clicked_at,omitempty"`
	RegisteredAt *time.Time `db:"registered_at" json:"registered_at,omitempty"`
	CompletedAt  *time.Time `db:"completed_at" json:"completed_at,omitempty"`

	// Rewards
	RewardGiven   bool       `db:"reward_given" json:"reward_given"`
	RewardType    *string    `db:"reward_type" json:"reward_type,omitempty"`
	RewardValue   *float64   `db:"reward_value" json:"reward_value,omitempty"`
	RewardGivenAt *time.Time `db:"reward_given_at" json:"reward_given_at,omitempty"`

	// Attribution
	UTMSource   *string `db:"utm_source" json:"utm_source,omitempty"`
	UTMCampaign *string `db:"utm_campaign" json:"utm_campaign,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (r *Referral) TableName() string {
	return "analytics.referrals"
}

func (r *Referral) PrimaryKey() interface{} {
	return r.ID
}

type ChurnPrediction struct {
	ID     string `db:"id" json:"id" pk:"true"`
	UserID string `db:"user_id" json:"user_id"`

	// Prediction
	ChurnProbability float64        `db:"churn_probability" json:"churn_probability"`
	ChurnRiskLevel   *ChurnRiskLevel `db:"churn_risk_level" json:"churn_risk_level,omitempty"`

	// Factors
	ContributingFactors json.RawMessage `db:"contributing_factors" json:"contributing_factors,omitempty"`

	// Timing
	PredictedChurnDate *time.Time `db:"predicted_churn_date" json:"predicted_churn_date,omitempty"`
	DaysUntilChurn     *int       `db:"days_until_churn" json:"days_until_churn,omitempty"`

	// Intervention
	InterventionRecommended *string    `db:"intervention_recommended" json:"intervention_recommended,omitempty"`
	InterventionSent        bool       `db:"intervention_sent" json:"intervention_sent"`
	InterventionSentAt      *time.Time `db:"intervention_sent_at" json:"intervention_sent_at,omitempty"`

	// Outcome
	ActuallyChurned *bool      `db:"actually_churned" json:"actually_churned,omitempty"`
	ChurnDate       *time.Time `db:"churn_date" json:"churn_date,omitempty"`

	PredictionDate  time.Time `db:"prediction_date" json:"prediction_date"`
	ModelVersion    *string   `db:"model_version" json:"model_version,omitempty"`
	ConfidenceScore *float64  `db:"confidence_score" json:"confidence_score,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

func (c *ChurnPrediction) TableName() string {
	return "analytics.churn_predictions"
}

func (c *ChurnPrediction) PrimaryKey() interface{} {
	return c.ID
}

type Attribution struct {
	ID     string `db:"id" json:"id" pk:"true"`
	UserID string `db:"user_id" json:"user_id"`

	// Conversion
	ConversionType  string     `db:"conversion_type" json:"conversion_type"`
	ConversionValue *float64   `db:"conversion_value" json:"conversion_value,omitempty"`
	ConvertedAt     time.Time  `db:"converted_at" json:"converted_at"`

	// Attribution
	AttributionModel string `db:"attribution_model" json:"attribution_model"`

	// Touchpoints
	FirstTouchSource   *string    `db:"first_touch_source" json:"first_touch_source,omitempty"`
	FirstTouchMedium   *string    `db:"first_touch_medium" json:"first_touch_medium,omitempty"`
	FirstTouchCampaign *string    `db:"first_touch_campaign" json:"first_touch_campaign,omitempty"`
	FirstTouchAt       *time.Time `db:"first_touch_at" json:"first_touch_at,omitempty"`

	LastTouchSource   *string    `db:"last_touch_source" json:"last_touch_source,omitempty"`
	LastTouchMedium   *string    `db:"last_touch_medium" json:"last_touch_medium,omitempty"`
	LastTouchCampaign *string    `db:"last_touch_campaign" json:"last_touch_campaign,omitempty"`
	LastTouchAt       *time.Time `db:"last_touch_at" json:"last_touch_at,omitempty"`

	// Journey
	TouchpointCount       int             `db:"touchpoint_count" json:"touchpoint_count"`
	JourneyDurationHours  *int            `db:"journey_duration_hours" json:"journey_duration_hours,omitempty"`
	Touchpoints           json.RawMessage `db:"touchpoints" json:"touchpoints,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (a *Attribution) TableName() string {
	return "analytics.attribution"
}

func (a *Attribution) PrimaryKey() interface{} {
	return a.ID
}

type HeatmapData struct {
	ID string `db:"id" json:"id" pk:"true"`

	ScreenName  string  `db:"screen_name" json:"screen_name"`
	ElementID   *string `db:"element_id" json:"element_id,omitempty"`
	ElementType *string `db:"element_type" json:"element_type,omitempty"`

	// Interaction
	InteractionType string `db:"interaction_type" json:"interaction_type"`

	// Position
	XCoordinate    *int `db:"x_coordinate" json:"x_coordinate,omitempty"`
	YCoordinate    *int `db:"y_coordinate" json:"y_coordinate,omitempty"`
	ViewportWidth  *int `db:"viewport_width" json:"viewport_width,omitempty"`
	ViewportHeight *int `db:"viewport_height" json:"viewport_height,omitempty"`

	// Frequency
	InteractionCount int `db:"interaction_count" json:"interaction_count"`

	// User context
	UserID    *string `db:"user_id" json:"user_id,omitempty"`
	SessionID *string `db:"session_id" json:"session_id,omitempty"`

	// Platform
	Platform   *string `db:"platform" json:"platform,omitempty"`
	DeviceType *string `db:"device_type" json:"device_type,omitempty"`

	Date      time.Time `db:"date" json:"date"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (h *HeatmapData) TableName() string {
	return "analytics.heatmap_data"
}

func (h *HeatmapData) PrimaryKey() interface{} {
	return h.ID
}