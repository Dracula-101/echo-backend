package domain

import (
	"time"

	db "shared/pkg/database/postgres/models"

	"github.com/google/uuid"
)

// DeliveryStatus represents message delivery status per recipient
type DeliveryStatus struct {
	ID          uuid.UUID
	MessageID   uuid.UUID
	RecipientID uuid.UUID
	Status      MessageStatus
	DeliveredAt *time.Time
	ReadAt      *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewDeliveryStatus(model db.DeliveryStatus, receipientId uuid.UUID) *DeliveryStatus {
	return &DeliveryStatus{
		ID:          uuid.MustParse(model.ID),
		MessageID:   uuid.MustParse(model.MessageID),
		RecipientID: receipientId,
		Status:      MessageStatus(model.Status),
		DeliveredAt: model.DeliveredAt,
		ReadAt:      model.ReadAt,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}

// IsRead checks if message has been read
func (d *DeliveryStatus) IsRead() bool {
	return d.Status == MessageStatusRead && d.ReadAt != nil
}

// IsDelivered checks if message has been delivered
func (d *DeliveryStatus) IsDelivered() bool {
	return d.Status == MessageStatusDelivered || d.Status == MessageStatusRead
}

// DeliveryReceipt contains information about message delivery
type DeliveryReceipt struct {
	UserID      uuid.UUID
	UserName    string
	DeliveredAt time.Time
}

func NewDeliveryReceipt(userID uuid.UUID, userName string, deliveredAt time.Time) DeliveryReceipt {
	return DeliveryReceipt{
		UserID:      userID,
		UserName:    userName,
		DeliveredAt: deliveredAt,
	}
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
