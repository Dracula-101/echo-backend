package models

import (
	"database/sql/driver"
	"fmt"
)

// ABTestStatus represents the status of an A/B test
type ABTestStatus string

const (
	ABTestStatusDraft     ABTestStatus = "draft"
	ABTestStatusActive    ABTestStatus = "active"
	ABTestStatusPaused    ABTestStatus = "paused"
	ABTestStatusCompleted ABTestStatus = "completed"
)

func (a ABTestStatus) IsValid() bool {
	switch a {
	case ABTestStatusDraft, ABTestStatusActive, ABTestStatusPaused, ABTestStatusCompleted:
		return true
	}
	return false
}

func (a ABTestStatus) Value() (driver.Value, error) {
	if !a.IsValid() {
		return nil, fmt.Errorf("invalid A/B test status: %s", a)
	}
	return string(a), nil
}

func (a *ABTestStatus) Scan(value interface{}) error {
	if value == nil {
		*a = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ABTestStatus: expected string, got %T", value)
	}
	*a = ABTestStatus(str)
	if !a.IsValid() {
		return fmt.Errorf("invalid A/B test status value: %s", str)
	}
	return nil
}

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	ErrorSeverityDebug    ErrorSeverity = "debug"
	ErrorSeverityInfo     ErrorSeverity = "info"
	ErrorSeverityWarning  ErrorSeverity = "warning"
	ErrorSeverityError    ErrorSeverity = "error"
	ErrorSeverityCritical ErrorSeverity = "critical"
)

func (e ErrorSeverity) IsValid() bool {
	switch e {
	case ErrorSeverityDebug, ErrorSeverityInfo, ErrorSeverityWarning,
		ErrorSeverityError, ErrorSeverityCritical:
		return true
	}
	return false
}

func (e ErrorSeverity) Value() (driver.Value, error) {
	if !e.IsValid() {
		return nil, fmt.Errorf("invalid error severity: %s", e)
	}
	return string(e), nil
}

func (e *ErrorSeverity) Scan(value interface{}) error {
	if value == nil {
		*e = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ErrorSeverity: expected string, got %T", value)
	}
	*e = ErrorSeverity(str)
	if !e.IsValid() {
		return fmt.Errorf("invalid error severity value: %s", str)
	}
	return nil
}

// RevenueEventStatus represents the status of a revenue event
type RevenueEventStatus string

const (
	RevenueEventStatusPending   RevenueEventStatus = "pending"
	RevenueEventStatusCompleted RevenueEventStatus = "completed"
	RevenueEventStatusRefunded  RevenueEventStatus = "refunded"
	RevenueEventStatusFailed    RevenueEventStatus = "failed"
)

func (r RevenueEventStatus) IsValid() bool {
	switch r {
	case RevenueEventStatusPending, RevenueEventStatusCompleted,
		RevenueEventStatusRefunded, RevenueEventStatusFailed:
		return true
	}
	return false
}

func (r RevenueEventStatus) Value() (driver.Value, error) {
	if !r.IsValid() {
		return nil, fmt.Errorf("invalid revenue event status: %s", r)
	}
	return string(r), nil
}

func (r *RevenueEventStatus) Scan(value interface{}) error {
	if value == nil {
		*r = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan RevenueEventStatus: expected string, got %T", value)
	}
	*r = RevenueEventStatus(str)
	if !r.IsValid() {
		return fmt.Errorf("invalid revenue event status value: %s", str)
	}
	return nil
}

// UserFeedbackStatus represents the status of user feedback
type UserFeedbackStatus string

const (
	UserFeedbackStatusNew        UserFeedbackStatus = "new"
	UserFeedbackStatusReviewed   UserFeedbackStatus = "reviewed"
	UserFeedbackStatusInProgress UserFeedbackStatus = "in_progress"
	UserFeedbackStatusResolved   UserFeedbackStatus = "resolved"
	UserFeedbackStatusClosed     UserFeedbackStatus = "closed"
)

func (u UserFeedbackStatus) IsValid() bool {
	switch u {
	case UserFeedbackStatusNew, UserFeedbackStatusReviewed, UserFeedbackStatusInProgress,
		UserFeedbackStatusResolved, UserFeedbackStatusClosed:
		return true
	}
	return false
}

func (u UserFeedbackStatus) Value() (driver.Value, error) {
	if !u.IsValid() {
		return nil, fmt.Errorf("invalid user feedback status: %s", u)
	}
	return string(u), nil
}

func (u *UserFeedbackStatus) Scan(value interface{}) error {
	if value == nil {
		*u = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan UserFeedbackStatus: expected string, got %T", value)
	}
	*u = UserFeedbackStatus(str)
	if !u.IsValid() {
		return fmt.Errorf("invalid user feedback status value: %s", str)
	}
	return nil
}

// UserFeedbackPriority represents the priority level of user feedback
type UserFeedbackPriority string

const (
	UserFeedbackPriorityLow    UserFeedbackPriority = "low"
	UserFeedbackPriorityMedium UserFeedbackPriority = "medium"
	UserFeedbackPriorityHigh   UserFeedbackPriority = "high"
)

func (u UserFeedbackPriority) IsValid() bool {
	switch u {
	case UserFeedbackPriorityLow, UserFeedbackPriorityMedium, UserFeedbackPriorityHigh:
		return true
	}
	return false
}

func (u UserFeedbackPriority) Value() (driver.Value, error) {
	if !u.IsValid() {
		return nil, fmt.Errorf("invalid user feedback priority: %s", u)
	}
	return string(u), nil
}

func (u *UserFeedbackPriority) Scan(value interface{}) error {
	if value == nil {
		*u = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan UserFeedbackPriority: expected string, got %T", value)
	}
	*u = UserFeedbackPriority(str)
	if !u.IsValid() {
		return fmt.Errorf("invalid user feedback priority value: %s", str)
	}
	return nil
}

// ReferralStatus represents the status of a referral
type ReferralStatus string

const (
	ReferralStatusPending   ReferralStatus = "pending"
	ReferralStatusCompleted ReferralStatus = "completed"
	ReferralStatusExpired   ReferralStatus = "expired"
)

func (r ReferralStatus) IsValid() bool {
	switch r {
	case ReferralStatusPending, ReferralStatusCompleted, ReferralStatusExpired:
		return true
	}
	return false
}

func (r ReferralStatus) Value() (driver.Value, error) {
	if !r.IsValid() {
		return nil, fmt.Errorf("invalid referral status: %s", r)
	}
	return string(r), nil
}

func (r *ReferralStatus) Scan(value interface{}) error {
	if value == nil {
		*r = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ReferralStatus: expected string, got %T", value)
	}
	*r = ReferralStatus(str)
	if !r.IsValid() {
		return fmt.Errorf("invalid referral status value: %s", str)
	}
	return nil
}

// ChurnRiskLevel represents the churn risk level
type ChurnRiskLevel string

const (
	ChurnRiskLevelLow      ChurnRiskLevel = "low"
	ChurnRiskLevelMedium   ChurnRiskLevel = "medium"
	ChurnRiskLevelHigh     ChurnRiskLevel = "high"
	ChurnRiskLevelCritical ChurnRiskLevel = "critical"
)

func (c ChurnRiskLevel) IsValid() bool {
	switch c {
	case ChurnRiskLevelLow, ChurnRiskLevelMedium, ChurnRiskLevelHigh, ChurnRiskLevelCritical:
		return true
	}
	return false
}

func (c ChurnRiskLevel) Value() (driver.Value, error) {
	if !c.IsValid() {
		return nil, fmt.Errorf("invalid churn risk level: %s", c)
	}
	return string(c), nil
}

func (c *ChurnRiskLevel) Scan(value interface{}) error {
	if value == nil {
		*c = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ChurnRiskLevel: expected string, got %T", value)
	}
	*c = ChurnRiskLevel(str)
	if !c.IsValid() {
		return fmt.Errorf("invalid churn risk level value: %s", str)
	}
	return nil
}

// WebhookDeliveryStatus represents the status of a webhook delivery
type WebhookDeliveryStatus string

const (
	WebhookDeliveryStatusPending  WebhookDeliveryStatus = "pending"
	WebhookDeliveryStatusSent     WebhookDeliveryStatus = "sent"
	WebhookDeliveryStatusFailed   WebhookDeliveryStatus = "failed"
	WebhookDeliveryStatusRetrying WebhookDeliveryStatus = "retrying"
)

func (w WebhookDeliveryStatus) IsValid() bool {
	switch w {
	case WebhookDeliveryStatusPending, WebhookDeliveryStatusSent,
		WebhookDeliveryStatusFailed, WebhookDeliveryStatusRetrying:
		return true
	}
	return false
}

func (w WebhookDeliveryStatus) Value() (driver.Value, error) {
	if !w.IsValid() {
		return nil, fmt.Errorf("invalid webhook delivery status: %s", w)
	}
	return string(w), nil
}

func (w *WebhookDeliveryStatus) Scan(value interface{}) error {
	if value == nil {
		*w = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan WebhookDeliveryStatus: expected string, got %T", value)
	}
	*w = WebhookDeliveryStatus(str)
	if !w.IsValid() {
		return fmt.Errorf("invalid webhook delivery status value: %s", str)
	}
	return nil
}

// ProcessingQueueStatus represents the status of a processing queue job
type ProcessingQueueStatus string

const (
	ProcessingQueueStatusQueued     ProcessingQueueStatus = "queued"
	ProcessingQueueStatusProcessing ProcessingQueueStatus = "processing"
	ProcessingQueueStatusCompleted  ProcessingQueueStatus = "completed"
	ProcessingQueueStatusFailed     ProcessingQueueStatus = "failed"
)

func (p ProcessingQueueStatus) IsValid() bool {
	switch p {
	case ProcessingQueueStatusQueued, ProcessingQueueStatusProcessing,
		ProcessingQueueStatusCompleted, ProcessingQueueStatusFailed:
		return true
	}
	return false
}

func (p ProcessingQueueStatus) Value() (driver.Value, error) {
	if !p.IsValid() {
		return nil, fmt.Errorf("invalid processing queue status: %s", p)
	}
	return string(p), nil
}

func (p *ProcessingQueueStatus) Scan(value interface{}) error {
	if value == nil {
		*p = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan ProcessingQueueStatus: expected string, got %T", value)
	}
	*p = ProcessingQueueStatus(str)
	if !p.IsValid() {
		return fmt.Errorf("invalid processing queue status value: %s", str)
	}
	return nil
}