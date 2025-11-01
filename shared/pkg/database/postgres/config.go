package postgres

import (
	"time"

	"shared/pkg/database"
)

func DefaultConfig() database.Config {
	return database.Config{
		Host:            "localhost",
		Port:            5432,
		User:            "postgres",
		Password:        "postgres",
		Database:        "postgres",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}
}
