package database

import (
	"context"
	"database/sql"
	"time"
)

type Model interface {
	TableName() string
	PrimaryKey() interface{}
}

type Database interface {
	Insert(ctx context.Context, model Model) (id *string, err *DBError)
	Upsert(ctx context.Context, model Model) *DBError
	FindByID(ctx context.Context, model Model, id interface{}) *DBError
	Update(ctx context.Context, model Model) *DBError
	Delete(ctx context.Context, model Model) *DBError
	HardDelete(ctx context.Context, model Model) *DBError

	FindOne(ctx context.Context, model Model, query string, args ...interface{}) *DBError
	FindMany(ctx context.Context, dest any, query string, args ...interface{}) *DBError
	FindOneAndUpdate(ctx context.Context, dest interface{}, query string, args ...interface{}) *DBError
	Exists(ctx context.Context, model Model, query string, args ...interface{}) (bool, *DBError)
	Count(ctx context.Context, model Model, query string, args ...interface{}) (int64, *DBError)

	Query(ctx context.Context, query string, args ...interface{}) (Rows, *DBError)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Exec(ctx context.Context, query string, args ...interface{}) (Result, *DBError)

	Begin(ctx context.Context) (Transaction, *DBError)
	BeginTx(ctx context.Context, opts *TxOptions) (Transaction, *DBError)
	WithTransaction(ctx context.Context, fn func(tx Transaction) *DBError) *DBError

	Close() *DBError
	Ping(ctx context.Context) *DBError
	Stats() Stats
}

type Transaction interface {
	Create(ctx context.Context, model Model) *DBError
	FindByID(ctx context.Context, model Model, id interface{}) *DBError
	Update(ctx context.Context, model Model) *DBError
	Delete(ctx context.Context, model Model) *DBError
	HardDelete(ctx context.Context, model Model) *DBError

	FindOne(ctx context.Context, model Model, query string, args ...interface{}) *DBError
	FindMany(ctx context.Context, dest interface{}, query string, args ...interface{}) *DBError

	Query(ctx context.Context, query string, args ...interface{}) (Rows, *DBError)
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Exec(ctx context.Context, query string, args ...interface{}) (Result, *DBError)

	Commit() *DBError
	Rollback() *DBError
}

type Rows interface {
	Next() bool
	Scan(dest ...interface{}) *DBError
	ScanAll(dest interface{}) *DBError
	Close() *DBError
	Err() *DBError
}

type Row interface {
	Scan(dest ...any) *DBError
	ScanModel(model Model) *DBError
}

type Result interface {
	LastInsertId() (int64, *DBError)
	RowsAffected() (int64, *DBError)
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
