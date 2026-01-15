package domain

import (
	"time"

	"github.com/google/uuid"
)

// ReadReceipt contains information about who read a message
type ReadReceipt struct {
	UserID   uuid.UUID
	UserName string
	ReadAt   time.Time
}
