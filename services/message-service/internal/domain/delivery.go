package domain

import (
	"time"

	"github.com/google/uuid"
)

// DeliveryReceipt contains information about message delivery
type DeliveryReceipt struct {
	UserID      uuid.UUID
	UserName    string
	DeliveredAt time.Time
}

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
