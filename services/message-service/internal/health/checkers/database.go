package checkers

import (
	"context"
	"fmt"
	"time"

	"echo-backend/services/message-service/internal/health"
	"shared/pkg/database"
)

type DatabaseChecker struct {
	db database.Database
}

func NewDatabaseChecker(db database.Database) *DatabaseChecker {
	return &DatabaseChecker{
		db: db,
	}
}

func (c *DatabaseChecker) Name() string {
	return "database"
}

func (c *DatabaseChecker) Check(ctx context.Context) health.CheckResult {
	start := time.Now()
	result := health.CheckResult{
		Status:      health.StatusHealthy,
		LastChecked: time.Now().Format(time.RFC3339),
	}

	if err := c.db.Ping(ctx); err != nil {
		result.Status = health.StatusUnhealthy
		result.Error = fmt.Sprintf("Database ping failed: %v", err)
		result.Message = "Unable to connect to PostgreSQL database"
		result.ResponseTime = float64(time.Since(start).Milliseconds())
		return result
	}

	stats := c.db.Stats()
	details := health.DatabaseDetails{
		Connected:       true,
		OpenConnections: stats.OpenConnections,
		IdleConnections: stats.Idle,
		MaxConnections:  stats.MaxOpenConnections,
		WaitCount:       stats.WaitCount,
		WaitDuration:    stats.WaitDuration.String(),
		MaxIdleTime:     fmt.Sprintf("%d", stats.MaxIdleTimeClosed),
		MaxLifetime:     fmt.Sprintf("%d", stats.MaxLifetimeClosed),
	}

	if stats.OpenConnections >= stats.MaxOpenConnections {
		result.Status = health.StatusDegraded
		result.Message = "Database connection pool is at maximum capacity"
	} else if stats.WaitCount > 100 {
		result.Status = health.StatusDegraded
		result.Message = fmt.Sprintf("High database connection wait count: %d", stats.WaitCount)
	} else {
		result.Message = "PostgreSQL database is healthy"
	}

	var dbVersion string
	queryCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	row := c.db.QueryRow(queryCtx, "SELECT version()")
	if err := row.Scan(&dbVersion); err != nil {
		result.Status = health.StatusDegraded
		result.Error = fmt.Sprintf("Query test failed: %v", err)
		result.Message = "Database is connected but queries are failing"
	}

	result.ResponseTime = float64(time.Since(start).Milliseconds())
	result.Details = map[string]interface{}{
		"database": details,
		"version":  dbVersion,
	}

	return result
}
