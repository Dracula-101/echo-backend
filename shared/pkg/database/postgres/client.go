package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"shared/pkg/database"
	"shared/pkg/logger"
	"shared/pkg/logger/adapter"
)

// Client implements the database.Database interface for PostgreSQL.
type Client struct {
	db  *sql.DB
	log *dbLogger
}

// New creates a new PostgreSQL client with the given configuration.
func New(config database.Config) (database.Database, error) {
	lgr, _ := adapter.NewZap(logger.Config{
		Level:      logger.GetLoggerLevel(),
		Format:     logger.FormatText,
		Output:     os.Stdout,
		TimeFormat: time.RFC3339,
		Service:    "postgres",
	})

	log := newDBLogger(lgr)

	dsn := buildDSN(config)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.ConnectionError("open", err)
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	configurePool(db, config)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.ConnectionError("ping", err)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("Database connection established",
		logger.String("host", config.Host),
		logger.Int("port", config.Port),
		logger.String("database", config.Database),
	)

	return &Client{db: db, log: log}, nil
}

// buildDSN constructs a PostgreSQL connection string.
func buildDSN(config database.Config) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode,
	)
}

// configurePool sets connection pool parameters.
func configurePool(db *sql.DB, config database.Config) {
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
}

// Insert creates a new record and returns the generated primary key.
func (c *Client) Insert(ctx context.Context, model database.Model) (*string, *database.DBError) {
	table := model.TableName()
	pkField := getPrimaryKeyField(model)

	fields, values := getInsertableFieldsAndValues(model, pkField)
	if len(fields) == 0 {
		return nil, c.log.NoFieldsError("Insert", table)
	}

	query := buildInsertQuery(table, fields, pkField)
	args := normalizeArgs(values)

	c.log.QueryStart("Insert", table, len(fields))

	var returnedID interface{}
	if err := c.db.QueryRowContext(ctx, query, args...).Scan(&returnedID); err != nil {
		return nil, c.log.QueryError("Insert", table, query, args, err)
	}

	if err := setPrimaryKeyValue(model, pkField, returnedID); err != nil {
		return nil, database.WrapDBError(err, database.CodeDBInternal, "failed to set primary key").
			WithDetail("table", table)
	}

	id := formatPrimaryKey(returnedID)
	c.log.QuerySuccess("Insert", table, id)
	return &id, nil
}

// Upsert inserts or updates a record based on primary key conflict.
func (c *Client) Upsert(ctx context.Context, model database.Model) *database.DBError {
	table := model.TableName()
	pkField := getPrimaryKeyField(model)
	conflictField := getConflictKeyField(model, pkField)

	fields, values := getInsertableFieldsAndValues(model, pkField)
	if len(fields) == 0 {
		return c.log.NoFieldsError("Upsert", table)
	}

	query := buildUpsertQuery(table, fields, pkField, conflictField)
	args := normalizeArgs(values)

	c.log.QueryStart("Upsert", table, len(fields))

	var returnedID interface{}
	if err := c.db.QueryRowContext(ctx, query, args...).Scan(&returnedID); err != nil {
		return c.log.QueryError("Upsert", table, query, args, err)
	}

	if err := setPrimaryKeyValue(model, pkField, returnedID); err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to set primary key").
			WithDetail("table", table)
	}

	c.log.QuerySuccess("Upsert", table, formatPrimaryKey(returnedID))
	return nil
}

// FindByID retrieves a record by its primary key.
func (c *Client) FindByID(ctx context.Context, model database.Model, id interface{}) *database.DBError {
	table := model.TableName()
	fields := getFields(model)
	if len(fields) == 0 {
		return c.log.NoFieldsError("FindByID", table)
	}

	query := buildSelectByIDQuery(table, fields, getPrimaryKeyField(model))

	c.log.QueryStart("FindByID", table, 0)

	row := c.db.QueryRowContext(ctx, query, id)
	if err := scanStruct(row, model); err != nil {
		return c.log.QueryError("FindByID", table, query, []interface{}{id}, err)
	}

	c.log.QuerySuccess("FindByID", table, fmt.Sprint(id))
	return nil
}

// Update modifies an existing record.
func (c *Client) Update(ctx context.Context, model database.Model) *database.DBError {
	table := model.TableName()
	pkField := getPrimaryKeyField(model)
	pk := model.PrimaryKey()

	setParts, updateValues := buildUpdateFields(model, pkField)
	if len(setParts) == 0 {
		return c.log.NoFieldsError("Update", table)
	}

	updateValues = append(updateValues, pk)
	query := buildUpdateQuery(table, setParts, pkField, len(updateValues))
	args := normalizeArgs(updateValues)

	c.log.QueryStart("Update", table, len(setParts))

	result, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return c.log.QueryError("Update", table, query, args, err)
	}

	if dbErr := checkRowsAffected(result, "Update", table, pk); dbErr != nil {
		return dbErr
	}

	c.log.QuerySuccess("Update", table, fmt.Sprint(pk))
	return nil
}

// Delete performs a soft delete by setting deleted_at timestamp.
func (c *Client) Delete(ctx context.Context, model database.Model) *database.DBError {
	table := model.TableName()
	pkField := getPrimaryKeyField(model)
	pk := model.PrimaryKey()

	query := fmt.Sprintf(
		"UPDATE %s SET deleted_at = $1 WHERE %s = $2 AND deleted_at IS NULL",
		table, pkField,
	)

	c.log.QueryStart("Delete", table, 0)

	result, err := c.db.ExecContext(ctx, query, time.Now(), pk)
	if err != nil {
		return c.log.QueryError("Delete", table, query, []interface{}{pk}, err)
	}

	if dbErr := checkRowsAffected(result, "Delete", table, pk); dbErr != nil {
		return dbErr
	}

	c.log.QuerySuccess("Delete", table, fmt.Sprint(pk))
	return nil
}

// HardDelete permanently removes a record from the database.
func (c *Client) HardDelete(ctx context.Context, model database.Model) *database.DBError {
	table := model.TableName()
	pkField := getPrimaryKeyField(model)
	pk := model.PrimaryKey()

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", table, pkField)

	c.log.QueryStart("HardDelete", table, 0)

	result, err := c.db.ExecContext(ctx, query, pk)
	if err != nil {
		return c.log.QueryError("HardDelete", table, query, []interface{}{pk}, err)
	}

	if dbErr := checkRowsAffected(result, "HardDelete", table, pk); dbErr != nil {
		return dbErr
	}

	c.log.QuerySuccess("HardDelete", table, fmt.Sprint(pk))
	return nil
}

// FindOne retrieves a single record matching the query.
func (c *Client) FindOne(ctx context.Context, model database.Model, query string, args ...interface{}) *database.DBError {
	table := model.TableName()
	nargs := normalizeArgs(args)

	c.log.QueryStart("FindOne", table, 0)

	row := c.db.QueryRowContext(ctx, query, nargs...)
	if err := scanStruct(row, model); err != nil {
		return c.log.QueryError("FindOne", table, query, nargs, err)
	}

	c.log.QuerySuccess("FindOne", table, "")
	return nil
}

// FindOneAndUpdate executes a query that modifies and returns a record.
func (c *Client) FindOneAndUpdate(ctx context.Context, dest interface{}, query string, args ...interface{}) *database.DBError {
	nargs := normalizeArgs(args)

	c.log.Debug("Executing FindOneAndUpdate", logger.String("query", query), logger.Any("args", args))

	row := c.db.QueryRowContext(ctx, query, nargs...)
	if err := scanStruct(row, dest); err != nil {
		return c.log.QueryError("FindOneAndUpdate", "", query, nargs, err)
	}

	return nil
}

// FindMany retrieves multiple records matching the query.
func (c *Client) FindMany(ctx context.Context, dest any, query string, args ...interface{}) *database.DBError {
	nargs := normalizeArgs(args)

	c.log.Debug("Executing FindMany", logger.String("query", query), logger.Any("args", args))

	rows, err := c.db.QueryContext(ctx, query, nargs...)
	if err != nil {
		return c.log.QueryError("FindMany", "", query, nargs, err)
	}
	defer rows.Close()

	if err := scanStructs(rows, dest, c.log.Logger); err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to scan results")
	}

	c.log.Debug("FindMany completed successfully")
	return nil
}

// Exists checks if a record matching the query exists.
func (c *Client) Exists(ctx context.Context, model database.Model, query string, args ...interface{}) (bool, *database.DBError) {
	nargs := normalizeArgs(args)

	c.log.Debug("Executing Exists check", logger.String("table", model.TableName()), logger.String("query", query), logger.Any("args", args))

	var exists bool
	if err := c.db.QueryRowContext(ctx, query, nargs...).Scan(&exists); err != nil {
		return false, c.log.QueryError("Exists", model.TableName(), query, nargs, err)
	}

	return exists, nil
}

// Count returns the number of records matching the query.
func (c *Client) Count(ctx context.Context, model database.Model, query string, args ...interface{}) (int64, *database.DBError) {
	nargs := normalizeArgs(args)

	c.log.Debug("Executing Count", logger.String("table", model.TableName()), logger.String("query", query), logger.Any("args", args))

	var count int64
	if err := c.db.QueryRowContext(ctx, query, nargs...).Scan(&count); err != nil {
		return 0, c.log.QueryError("Count", model.TableName(), query, nargs, err)
	}

	return count, nil
}

// Query executes a raw query and returns the rows.
func (c *Client) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, *database.DBError) {
	nargs := normalizeArgs(args)

	c.log.Debug("Executing raw Query", logger.String("query", query), logger.Any("args", args))

	rows, err := c.db.QueryContext(ctx, query, nargs...)
	if err != nil {
		return nil, c.log.QueryError("Query", "", query, nargs, err)
	}

	return &RowsWrapper{rows: rows, log: c.log.Logger}, nil
}

// QueryRow executes a query expected to return a single row.
func (c *Client) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	nargs := normalizeArgs(args)

	c.log.Debug("Executing QueryRow", logger.String("query", query), logger.Any("args", args))

	return &RowWrapper{row: c.db.QueryRowContext(ctx, query, nargs...), log: c.log.Logger}
}

// Exec executes a query that doesn't return rows.
func (c *Client) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, *database.DBError) {
	nargs := normalizeArgs(args)

	c.log.Debug("Executing Exec", logger.String("query", query), logger.Any("args", args))

	result, err := c.db.ExecContext(ctx, query, nargs...)
	if err != nil {
		return nil, c.log.QueryError("Exec", "", query, nargs, err)
	}

	return &ResultWrapper{result: result}, nil
}

// Begin starts a new transaction.
func (c *Client) Begin(ctx context.Context) (database.Transaction, *database.DBError) {
	c.log.Debug("Beginning transaction")

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		c.log.Error("Failed to begin transaction", logger.Error(err))
		return nil, database.WrapDBError(err, database.CodeDBInternal, "failed to begin transaction")
	}

	return newTransaction(tx, c.log), nil
}

// BeginTx starts a new transaction with the specified options.
func (c *Client) BeginTx(ctx context.Context, opts *database.TxOptions) (database.Transaction, *database.DBError) {
	c.log.Debug("Beginning transaction with options")

	sqlOpts := &sql.TxOptions{}
	if opts != nil {
		sqlOpts.Isolation = opts.Isolation
		sqlOpts.ReadOnly = opts.ReadOnly
	}

	tx, err := c.db.BeginTx(ctx, sqlOpts)
	if err != nil {
		c.log.Error("Failed to begin transaction with options", logger.Error(err))
		return nil, database.WrapDBError(err, database.CodeDBInternal, "failed to begin transaction")
	}

	return newTransaction(tx, c.log), nil
}

// WithTransaction executes a function within a transaction.
func (c *Client) WithTransaction(ctx context.Context, fn func(tx database.Transaction) *database.DBError) *database.DBError {
	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			c.log.Error("Transaction panic recovered", logger.Any("panic", p))
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			c.log.Error("Failed to rollback after error", logger.Error(rbErr))
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		c.log.Error("Failed to commit transaction", logger.Error(err))
		return err
	}

	c.log.Debug("Transaction completed successfully")
	return nil
}

// Close closes the database connection.
func (c *Client) Close() *database.DBError {
	c.log.Info("Closing database connection")

	if err := c.db.Close(); err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to close database")
	}

	return nil
}

// Ping verifies the database connection is alive.
func (c *Client) Ping(ctx context.Context) *database.DBError {
	if err := c.db.PingContext(ctx); err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to ping database")
	}
	return nil
}

// Stats returns database connection pool statistics.
func (c *Client) Stats() database.Stats {
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

// Query builder helpers

func buildInsertQuery(table string, fields []string, pkField string) string {
	placeholders := make([]string, len(fields))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
		table,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
		pkField,
	)
}

func buildUpsertQuery(table string, fields []string, pkField, conflictField string) string {
	placeholders := make([]string, len(fields))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	updateParts := make([]string, 0, len(fields))
	for _, field := range fields {
		if field != pkField && field != conflictField && field != "created_at" {
			updateParts = append(updateParts, fmt.Sprintf("%s = EXCLUDED.%s", field, field))
		}
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s RETURNING %s",
		table,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
		conflictField,
		strings.Join(updateParts, ", "),
		pkField,
	)
}

func buildSelectByIDQuery(table string, fields []string, pkField string) string {
	return fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = $1",
		strings.Join(fields, ", "),
		table,
		pkField,
	)
}

func buildUpdateQuery(table string, setParts []string, pkField string, paramCount int) string {
	return fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d",
		table,
		strings.Join(setParts, ", "),
		pkField,
		paramCount,
	)
}

func buildUpdateFields(model database.Model, pkField string) ([]string, []interface{}) {
	fields, values := getFieldsAndValues(model)

	setParts := make([]string, 0, len(fields))
	updateValues := make([]interface{}, 0, len(values))

	for i, field := range fields {
		if field == pkField || field == "created_at" {
			continue
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, len(setParts)+1))
		updateValues = append(updateValues, values[i])
	}

	return setParts, updateValues
}

func checkRowsAffected(result sql.Result, operation, table string, pk interface{}) *database.DBError {
	rows, err := result.RowsAffected()
	if err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to get rows affected").
			WithDetail("table", table)
	}

	if rows == 0 {
		msg := "record not found"
		if operation == "Delete" {
			msg = "record not found or already deleted"
		}
		return database.NewDBError(database.CodeDBInternal, msg).
			WithDetail("operation", operation).
			WithDetail("table", table).
			WithDetail("primary_key", pk)
	}

	return nil
}
