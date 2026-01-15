package repo

import (
	"shared/pkg/cache"
	"shared/pkg/database"
)

type MessageRepository interface {
}

type messageRepository struct {
	db    database.Database
	cache cache.Cache
}

func NewMessageRepository(db database.Database, cache cache.Cache) MessageRepository {
	return &messageRepository{db: db, cache: cache}
}
