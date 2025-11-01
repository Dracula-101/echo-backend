package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"

	"shared/pkg/database"
	"shared/pkg/logger"
	"shared/pkg/logger/adapter"
)

type client struct {
	db     *sql.DB
	logger logger.Logger
}

func New(config database.Config) (database.Database, error) {
	lgr, _ := adapter.NewZap(logger.Config{
		Level:      logger.GetLoggerLevel(),
		Format:     logger.FormatText,
		Output:     os.Stdout,
		TimeFormat: time.RFC3339,
		Service:    "postgres-client",
	})

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode,
	)

	lgr.Debug("Connecting to database",
		logger.String("host", config.Host),
		logger.Int("port", config.Port),
		logger.String("user", config.User),
		logger.String("database", config.Database),
		logger.String("sslmode", config.SSLMode))

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		lgr.Error("Failed to open database connection",
			logger.String("database", config.Database),
			logger.String("host", config.Host),
			logger.Int("port", config.Port),
			logger.Error(err))
		return nil, err
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		lgr.Error("Failed to ping database",
			logger.String("database", config.Database),
			logger.String("host", config.Host),
			logger.Int("port", config.Port),
			logger.Error(err))
		return nil, err
	}

	lgr.Info("Successfully connected to database",
		logger.String("database", config.Database),
		logger.String("host", config.Host))

	return &client{
		db:     db,
		logger: lgr,
	}, nil
}

func (c *client) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	c.logger.Debug("Executing query", logger.String("query", query), logger.Any("args", args))
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &rowsWrapper{rows: rows}, nil
}

func (c *client) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	c.logger.Debug("Executing query", logger.String("query", query), logger.Any("args", args))
	return &rowWrapper{row: c.db.QueryRowContext(ctx, query, args...)}
}

func (c *client) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	c.logger.Debug("Executing exec", logger.String("query", query), logger.Any("args", args))
	result, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &resultWrapper{result: result}, nil
}

func (c *client) Begin(ctx context.Context) (database.Transaction, error) {
	c.logger.Debug("Beginning transaction")
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &transactionWrapper{tx: tx}, nil
}

func (c *client) BeginTx(ctx context.Context, opts *database.TxOptions) (database.Transaction, error) {
	c.logger.Debug("Beginning transaction with options", logger.Any("options", opts))
	sqlOpts := &sql.TxOptions{}
	if opts != nil {
		sqlOpts.Isolation = opts.Isolation
		sqlOpts.ReadOnly = opts.ReadOnly
	}

	tx, err := c.db.BeginTx(ctx, sqlOpts)
	if err != nil {
		return nil, err
	}
	return &transactionWrapper{tx: tx}, nil
}

func (c *client) Close() error {
	c.logger.Debug("Closing database connection")
	return c.db.Close()
}

func (c *client) Ping(ctx context.Context) error {
	c.logger.Debug("Pinging database")
	return c.db.PingContext(ctx)
}

func (c *client) Stats() database.Stats {
	stats := c.db.Stats()
	return database.Stats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}
}

type transactionWrapper struct {
	tx *sql.Tx
}

func (t *transactionWrapper) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &rowsWrapper{rows: rows}, nil
}

func (t *transactionWrapper) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	return &rowWrapper{row: t.tx.QueryRowContext(ctx, query, args...)}
}

func (t *transactionWrapper) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &resultWrapper{result: result}, nil
}

func (t *transactionWrapper) Commit() error {
	return t.tx.Commit()
}

func (t *transactionWrapper) Rollback() error {
	return t.tx.Rollback()
}

type rowsWrapper struct {
	rows *sql.Rows
}

func (r *rowsWrapper) Next() bool {
	return r.rows.Next()
}

func (r *rowsWrapper) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *rowsWrapper) Close() error {
	return r.rows.Close()
}

func (r *rowsWrapper) Err() error {
	return r.rows.Err()
}

type rowWrapper struct {
	row *sql.Row
}

func (r *rowWrapper) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}

type resultWrapper struct {
	result sql.Result
}

func (r *resultWrapper) LastInsertId() (int64, error) {
	return r.result.LastInsertId()
}

func (r *resultWrapper) RowsAffected() (int64, error) {
	return r.result.RowsAffected()
}
