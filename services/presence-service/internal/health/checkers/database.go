package checkers

import (
	"context"
	"presence-service/internal/health"

	"shared/pkg/database"
)

type DatabaseChecker struct {
	db database.Database
}

func NewDatabaseChecker(db database.Database) health.Checker {
	return &DatabaseChecker{db: db}
}

func (c *DatabaseChecker) Name() string {
	return "database"
}

func (c *DatabaseChecker) Check(ctx context.Context) health.CheckResult {
	if err := c.db.Ping(ctx); err != nil {
		return health.CheckResult{
			Name:    c.Name(),
			Status:  health.StatusUnhealthy,
			Message: "Database connection failed",
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}

	return health.CheckResult{
		Name:    c.Name(),
		Status:  health.StatusHealthy,
		Message: "Database is responsive",
	}
}
