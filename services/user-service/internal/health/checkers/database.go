package checkers

import (
	"context"
	"database/sql"
	"fmt"
	"shared/pkg/database"
	"time"
	"user-service/internal/health"
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

	// Test database connectivity with a ping
	if err := c.db.Ping(ctx); err != nil {
		result.Status = health.StatusUnhealthy
		result.Error = fmt.Sprintf("Database ping failed: %v", err)
		result.Message = "Unable to connect to PostgreSQL database"
		result.ResponseTime = float64(time.Since(start).Milliseconds())
		return result
	}

	// Get database statistics
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

	// Check connection pool health
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

type CustomQueryChecker struct {
	db    *sql.DB
	query string
	name  string
}

func NewCustomQueryChecker(db *sql.DB, name, query string) *CustomQueryChecker {
	return &CustomQueryChecker{
		db:    db,
		query: query,
		name:  name,
	}
}

func (c *CustomQueryChecker) Name() string {
	return c.name
}

func (c *CustomQueryChecker) Check(ctx context.Context) health.CheckResult {
	start := time.Now()
	result := health.CheckResult{
		Status:      health.StatusHealthy,
		LastChecked: time.Now().Format(time.RFC3339),
	}

	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := c.db.ExecContext(queryCtx, c.query)
	if err != nil {
		result.Status = health.StatusUnhealthy
		result.Error = fmt.Sprintf("Custom query failed: %v", err)
		result.Message = "Custom database check failed"
	} else {
		result.Message = "Custom database check passed"
	}

	result.ResponseTime = float64(time.Since(start).Milliseconds())
	return result
}
