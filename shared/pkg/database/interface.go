package database

import (
	"context"
	"database/sql"
	"time"

	pkgErrors "shared/pkg/errors"
)

type Model interface {
	TableName() string
	PrimaryKey() interface{}
}

type Database interface {
	Create(ctx context.Context, model Model) (id *string, err error)
	FindByID(ctx context.Context, model Model, id interface{}) pkgErrors.AppError
	Update(ctx context.Context, model Model) pkgErrors.AppError
	Delete(ctx context.Context, model Model) pkgErrors.AppError
	HardDelete(ctx context.Context, model Model) pkgErrors.AppError

	FindOne(ctx context.Context, model Model, query string, args ...interface{}) pkgErrors.AppError
	FindMany(ctx context.Context, dest interface{}, query string, args ...interface{}) pkgErrors.AppError
	Exists(ctx context.Context, model Model, query string, args ...interface{}) (bool, error)
	Count(ctx context.Context, model Model, query string, args ...interface{}) (int64, error)

	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)

	Begin(ctx context.Context) (Transaction, error)
	BeginTx(ctx context.Context, opts *TxOptions) (Transaction, error)
	WithTransaction(ctx context.Context, fn func(tx Transaction) error) error

	Close() error
	Ping(ctx context.Context) pkgErrors.AppError
	Stats() Stats
}

type Transaction interface {
	Create(ctx context.Context, model Model) pkgErrors.AppError
	FindByID(ctx context.Context, model Model, id interface{}) pkgErrors.AppError
	Update(ctx context.Context, model Model) pkgErrors.AppError
	Delete(ctx context.Context, model Model) pkgErrors.AppError
	HardDelete(ctx context.Context, model Model) pkgErrors.AppError

	FindOne(ctx context.Context, model Model, query string, args ...interface{}) pkgErrors.AppError
	FindMany(ctx context.Context, dest interface{}, query string, args ...interface{}) pkgErrors.AppError

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
	ScanOne(model Model) pkgErrors.AppError
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

//
