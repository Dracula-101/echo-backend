package database

import (
	"context"
	"database/sql"
	"time"
)

type Database interface {
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
	Begin(ctx context.Context) (Transaction, error)
	BeginTx(ctx context.Context, opts *TxOptions) (Transaction, error)
	Close() error
	Ping(ctx context.Context) error
	Stats() Stats
}

type Transaction interface {
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
	Commit() error
	Rollback() error
}

type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close() error
	Err() error
}

type Row interface {
	Scan(dest ...interface{}) error
}

type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

type TxOptions struct {
	Isolation sql.IsolationLevel
	ReadOnly  bool
}

type Stats struct {
	MaxOpenConnections int
	OpenConnections    int
	InUse              int
	Idle               int
	WaitCount          int64
	WaitDuration       time.Duration
	MaxIdleClosed      int64
	MaxIdleTimeClosed  int64
	MaxLifetimeClosed  int64
}

type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}
