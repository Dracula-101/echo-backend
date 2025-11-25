package checkers

import (
	"context"
	"ws-service/internal/health"

	"shared/pkg/database"
)

type DatabaseChecker struct {
	db database.Database
}

func NewDatabaseChecker(db database.Database) *DatabaseChecker {
	return &DatabaseChecker{db: db}
}

func (c *DatabaseChecker) Name() string {
	return "database"
}

func (c *DatabaseChecker) Check(ctx context.Context) (health.Status, string) {
	if err := c.db.Ping(ctx); err != nil {
		return health.StatusUnhealthy, "Database connection failed: " + err.Error()
	}
	return health.StatusHealthy, "Database connection successful"
}
