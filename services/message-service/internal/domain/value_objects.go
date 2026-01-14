package domain

import (
	"time"

	"github.com/google/uuid"
)

// DeliveryStatusSummary contains aggregated delivery information
type DeliveryStatusSummary struct {
	MessageID       uuid.UUID
	TotalRecipients int
	DeliveredCount  int
	ReadCount       int
	PendingCount    int
	FailedCount     int
	ReadBy          []ReadReceipt
	DeliveredBy     []DeliveryReceipt
}

// ReadReceipt contains information about who read a message
type ReadReceipt struct {
	UserID   uuid.UUID
	UserName string
	ReadAt   time.Time
}

// DeliveryReceipt contains information about message delivery
type DeliveryReceipt struct {
	UserID      uuid.UUID
	UserName    string
	DeliveredAt time.Time
}

// ============================================================================
// Pagination
// ============================================================================

// PaginationInfo contains pagination metadata
type PaginationInfo struct {
	Limit      int
	Offset     int
	TotalCount int
	HasMore    bool
	NextCursor *string
	PrevCursor *string
}
