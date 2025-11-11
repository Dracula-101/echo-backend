package models

import (
	"database/sql/driver"
	"fmt"
)

// NotificationDeliveryStatus represents the delivery status of a notification
type NotificationDeliveryStatus string

const (
	NotificationDeliveryStatusPending   NotificationDeliveryStatus = "pending"
	NotificationDeliveryStatusSent      NotificationDeliveryStatus = "sent"
	NotificationDeliveryStatusDelivered NotificationDeliveryStatus = "delivered"
	NotificationDeliveryStatusFailed    NotificationDeliveryStatus = "failed"
)

func (n NotificationDeliveryStatus) IsValid() bool {
	switch n {
	case NotificationDeliveryStatusPending, NotificationDeliveryStatusSent,
		NotificationDeliveryStatusDelivered, NotificationDeliveryStatusFailed:
		return true
	}
	return false
}

func (n NotificationDeliveryStatus) Value() (driver.Value, error) {
	if !n.IsValid() {
		return nil, fmt.Errorf("invalid notification delivery status: %s", n)
	}
	return string(n), nil
}

func (n *NotificationDeliveryStatus) Scan(value interface{}) error {
	if value == nil {
		*n = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan NotificationDeliveryStatus: expected string, got %T", value)
	}
	*n = NotificationDeliveryStatus(str)
	if !n.IsValid() {
		return fmt.Errorf("invalid notification delivery status value: %s", str)
	}
	return nil
}

// NotificationPriority represents the priority level of a notification
type NotificationPriority string

const (
	NotificationPriorityLow    NotificationPriority = "low"
	NotificationPriorityNormal NotificationPriority = "normal"
	NotificationPriorityHigh   NotificationPriority = "high"
	NotificationPriorityUrgent NotificationPriority = "urgent"
)

func (n NotificationPriority) IsValid() bool {
	switch n {
	case NotificationPriorityLow, NotificationPriorityNormal,
		NotificationPriorityHigh, NotificationPriorityUrgent:
		return true
	}
	return false
}

func (n NotificationPriority) Value() (driver.Value, error) {
	if !n.IsValid() {
		return nil, fmt.Errorf("invalid notification priority: %s", n)
	}
	return string(n), nil
}

func (n *NotificationPriority) Scan(value interface{}) error {
	if value == nil {
		*n = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan NotificationPriority: expected string, got %T", value)
	}
	*n = NotificationPriority(str)
	if !n.IsValid() {
		return fmt.Errorf("invalid notification priority value: %s", str)
	}
	return nil
}

// PushDeliveryStatus represents the status of a push notification delivery
type PushDeliveryStatus string

const (
	PushDeliveryStatusPending   PushDeliveryStatus = "pending"
	PushDeliveryStatusSent      PushDeliveryStatus = "sent"
	PushDeliveryStatusDelivered PushDeliveryStatus = "delivered"
	PushDeliveryStatusFailed    PushDeliveryStatus = "failed"
	PushDeliveryStatusExpired   PushDeliveryStatus = "expired"
)

func (p PushDeliveryStatus) IsValid() bool {
	switch p {
	case PushDeliveryStatusPending, PushDeliveryStatusSent, PushDeliveryStatusDelivered,
		PushDeliveryStatusFailed, PushDeliveryStatusExpired:
		return true
	}
	return false
}

func (p PushDeliveryStatus) Value() (driver.Value, error) {
	if !p.IsValid() {
		return nil, fmt.Errorf("invalid push delivery status: %s", p)
	}
	return string(p), nil
}

func (p *PushDeliveryStatus) Scan(value interface{}) error {
	if value == nil {
		*p = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan PushDeliveryStatus: expected string, got %T", value)
	}
	*p = PushDeliveryStatus(str)
	if !p.IsValid() {
		return fmt.Errorf("invalid push delivery status value: %s", str)
	}
	return nil
}

// EmailStatus represents the status of an email notification
type EmailStatus string

const (
	EmailStatusPending   EmailStatus = "pending"
	EmailStatusSent      EmailStatus = "sent"
	EmailStatusDelivered EmailStatus = "delivered"
	EmailStatusBounced   EmailStatus = "bounced"
	EmailStatusFailed    EmailStatus = "failed"
)

func (e EmailStatus) IsValid() bool {
	switch e {
	case EmailStatusPending, EmailStatusSent, EmailStatusDelivered,
		EmailStatusBounced, EmailStatusFailed:
		return true
	}
	return false
}

func (e EmailStatus) Value() (driver.Value, error) {
	if !e.IsValid() {
		return nil, fmt.Errorf("invalid email status: %s", e)
	}
	return string(e), nil
}

func (e *EmailStatus) Scan(value interface{}) error {
	if value == nil {
		*e = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan EmailStatus: expected string, got %T", value)
	}
	*e = EmailStatus(str)
	if !e.IsValid() {
		return fmt.Errorf("invalid email status value: %s", str)
	}
	return nil
}

// SMSStatus represents the status of an SMS notification
type SMSStatus string

const (
	SMSStatusPending     SMSStatus = "pending"
	SMSStatusSent        SMSStatus = "sent"
	SMSStatusDelivered   SMSStatus = "delivered"
	SMSStatusFailed      SMSStatus = "failed"
	SMSStatusUndelivered SMSStatus = "undelivered"
)

func (s SMSStatus) IsValid() bool {
	switch s {
	case SMSStatusPending, SMSStatusSent, SMSStatusDelivered,
		SMSStatusFailed, SMSStatusUndelivered:
		return true
	}
	return false
}

func (s SMSStatus) Value() (driver.Value, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("invalid SMS status: %s", s)
	}
	return string(s), nil
}

func (s *SMSStatus) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan SMSStatus: expected string, got %T", value)
	}
	*s = SMSStatus(str)
	if !s.IsValid() {
		return fmt.Errorf("invalid SMS status value: %s", str)
	}
	return nil
}

// NotificationBatchType represents the type of notification batch
type NotificationBatchType string

const (
	NotificationBatchTypeBroadcast  NotificationBatchType = "broadcast"
	NotificationBatchTypeSegment    NotificationBatchType = "segment"
	NotificationBatchTypeIndividual NotificationBatchType = "individual"
)

func (n NotificationBatchType) IsValid() bool {
	switch n {
	case NotificationBatchTypeBroadcast, NotificationBatchTypeSegment, NotificationBatchTypeIndividual:
		return true
	}
	return false
}

func (n NotificationBatchType) Value() (driver.Value, error) {
	if !n.IsValid() {
		return nil, fmt.Errorf("invalid notification batch type: %s", n)
	}
	return string(n), nil
}

func (n *NotificationBatchType) Scan(value interface{}) error {
	if value == nil {
		*n = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan NotificationBatchType: expected string, got %T", value)
	}
	*n = NotificationBatchType(str)
	if !n.IsValid() {
		return fmt.Errorf("invalid notification batch type value: %s", str)
	}
	return nil
}

// NotificationBatchStatus represents the status of a notification batch
type NotificationBatchStatus string

const (
	NotificationBatchStatusPending    NotificationBatchStatus = "pending"
	NotificationBatchStatusProcessing NotificationBatchStatus = "processing"
	NotificationBatchStatusCompleted  NotificationBatchStatus = "completed"
	NotificationBatchStatusFailed     NotificationBatchStatus = "failed"
	NotificationBatchStatusCancelled  NotificationBatchStatus = "cancelled"
)

func (n NotificationBatchStatus) IsValid() bool {
	switch n {
	case NotificationBatchStatusPending, NotificationBatchStatusProcessing,
		NotificationBatchStatusCompleted, NotificationBatchStatusFailed,
		NotificationBatchStatusCancelled:
		return true
	}
	return false
}

func (n NotificationBatchStatus) Value() (driver.Value, error) {
	if !n.IsValid() {
		return nil, fmt.Errorf("invalid notification batch status: %s", n)
	}
	return string(n), nil
}

func (n *NotificationBatchStatus) Scan(value interface{}) error {
	if value == nil {
		*n = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan NotificationBatchStatus: expected string, got %T", value)
	}
	*n = NotificationBatchStatus(str)
	if !n.IsValid() {
		return fmt.Errorf("invalid notification batch status value: %s", str)
	}
	return nil
}

// AnnouncementSeverity represents the severity level of an announcement
type AnnouncementSeverity string

const (
	AnnouncementSeverityInfo    AnnouncementSeverity = "info"
	AnnouncementSeverityWarning AnnouncementSeverity = "warning"
	AnnouncementSeverityError   AnnouncementSeverity = "error"
	AnnouncementSeveritySuccess AnnouncementSeverity = "success"
)

func (a AnnouncementSeverity) IsValid() bool {
	switch a {
	case AnnouncementSeverityInfo, AnnouncementSeverityWarning,
		AnnouncementSeverityError, AnnouncementSeveritySuccess:
		return true
	}
	return false
}

func (a AnnouncementSeverity) Value() (driver.Value, error) {
	if !a.IsValid() {
		return nil, fmt.Errorf("invalid announcement severity: %s", a)
	}
	return string(a), nil
}

func (a *AnnouncementSeverity) Scan(value interface{}) error {
	if value == nil {
		*a = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan AnnouncementSeverity: expected string, got %T", value)
	}
	*a = AnnouncementSeverity(str)
	if !a.IsValid() {
		return fmt.Errorf("invalid announcement severity value: %s", str)
	}
	return nil
}

// AnnouncementDisplayFrequency represents how often an announcement should be displayed
type AnnouncementDisplayFrequency string

const (
	AnnouncementDisplayFrequencyOnce    AnnouncementDisplayFrequency = "once"
	AnnouncementDisplayFrequencyDaily   AnnouncementDisplayFrequency = "daily"
	AnnouncementDisplayFrequencySession AnnouncementDisplayFrequency = "session"
	AnnouncementDisplayFrequencyAlways  AnnouncementDisplayFrequency = "always"
)

func (a AnnouncementDisplayFrequency) IsValid() bool {
	switch a {
	case AnnouncementDisplayFrequencyOnce, AnnouncementDisplayFrequencyDaily,
		AnnouncementDisplayFrequencySession, AnnouncementDisplayFrequencyAlways:
		return true
	}
	return false
}

func (a AnnouncementDisplayFrequency) Value() (driver.Value, error) {
	if !a.IsValid() {
		return nil, fmt.Errorf("invalid announcement display frequency: %s", a)
	}
	return string(a), nil
}

func (a *AnnouncementDisplayFrequency) Scan(value interface{}) error {
	if value == nil {
		*a = ""
		return nil
	}
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to scan AnnouncementDisplayFrequency: expected string, got %T", value)
	}
	*a = AnnouncementDisplayFrequency(str)
	if !a.IsValid() {
		return fmt.Errorf("invalid announcement display frequency value: %s", str)
	}
	return nil
}
