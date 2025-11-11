package checkers

import (
	"context"
	"time"

	"media-service/internal/health"

	"shared/pkg/database"
)

// DatabaseChecker checks database connectivity
type DatabaseChecker struct {
	db database.Database
}

// NewDatabaseChecker creates a new database health checker
func NewDatabaseChecker(db database.Database) *DatabaseChecker {
	return &DatabaseChecker{
		db: db,
	}
}

// Name returns the name of this checker
func (c *DatabaseChecker) Name() string {
	return "database"
}

// Check performs the database health check
func (c *DatabaseChecker) Check(ctx context.Context) health.CheckResult {
	start := time.Now()

	if err := c.db.Ping(ctx); err != nil {
		return health.CheckResult{
			Status:    health.StatusUnhealthy,
			Timestamp: time.Now(),
			Error:     err.Error(),
			Details: map[string]interface{}{
				"response_time_ms": time.Since(start).Milliseconds(),
			},
		}
	}

	stats := c.db.Stats()

	return health.CheckResult{
		Status:    health.StatusHealthy,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"response_time_ms":     time.Since(start).Milliseconds(),
			"open_connections":     stats.OpenConnections,
			"in_use":               stats.InUse,
			"idle":                 stats.Idle,
			"max_open_connections": stats.MaxOpenConnections,
		},
	}
}
