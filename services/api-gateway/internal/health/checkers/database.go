package checkers

import (
	"context"
	"database/sql"
	"time"

	"echo-backend/services/api-gateway/internal/health"
)

type DatabaseChecker struct {
	db *sql.DB
}

func NewDatabaseChecker(db *sql.DB) *DatabaseChecker {
	return &DatabaseChecker{db: db}
}

func (c *DatabaseChecker) Name() string {
	return "database"
}

func (c *DatabaseChecker) Check(ctx context.Context) health.CheckResult {
	start := time.Now()

	if err := c.db.PingContext(ctx); err != nil {
		return health.CheckResult{
			Status:       health.StatusUnhealthy,
			Message:      "Database connection failed",
			Error:        err.Error(),
			ResponseTime: time.Since(start).Seconds() * 1000,
			LastChecked:  time.Now().Format(time.RFC3339),
		}
	}

	stats := c.db.Stats()
	if stats.OpenConnections >= stats.MaxOpenConnections {
		return health.CheckResult{
			Status:       health.StatusDegraded,
			Message:      "Connection pool near capacity",
			ResponseTime: time.Since(start).Seconds() * 1000,
			LastChecked:  time.Now().Format(time.RFC3339),
		}
	}

	return health.CheckResult{
		Status:       health.StatusHealthy,
		Message:      "Database connection OK",
		ResponseTime: time.Since(start).Seconds() * 1000,
		LastChecked:  time.Now().Format(time.RFC3339),
	}
}
