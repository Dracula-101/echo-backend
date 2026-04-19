package domain

import (
	"time"

	dbModels "shared/pkg/database/postgres/models"
)

type BlockType string

const (
	BlockTypeUser   BlockType = "user"
	BlockTypeDomain BlockType = "domain"
	BlockTypeIP     BlockType = "ip"
)

type BlockedUser struct {
	ID            string
	UserID        string
	BlockedUserID string
	BlockType     BlockType
	Reason        *string
	BlockedAt     time.Time
	UnblockedAt   *time.Time
}

func NewBlockedUser(model dbModels.BlockedUser) *BlockedUser {
	return &BlockedUser{
		ID:            model.ID,
		UserID:        model.UserID,
		BlockedUserID: model.BlockedUserID,
		BlockType:     BlockType(model.BlockType),
		Reason:        model.BlockReason,
		BlockedAt:     model.BlockedAt,
		UnblockedAt:   model.UnblockedAt,
	}
}
