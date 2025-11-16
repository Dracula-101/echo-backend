package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"shared/pkg/database"
	"shared/pkg/logger"
	"shared/pkg/logger/adapter"

	"github.com/lib/pq"
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

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		lgr.Error("Failed to open database", logger.Error(err))
		return nil, err
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		lgr.Error("Failed to ping database", logger.Error(err))
		return nil, err
	}

	lgr.Info("Connected to database")

	return &client{
		db:     db,
		logger: lgr,
	}, nil
}

func (c *client) Create(ctx context.Context, model database.Model) (*string, error) {
	fields, values := getFieldsAndValues(model)
	if len(fields) == 0 {
		return nil, database.NewDBError(database.CodeDBInternal, "no db tags found in model").
			WithDetail("table", model.TableName())
	}

	pkField := getPrimaryKeyField(model)
	filteredFields := []string{}
	filteredValues := []interface{}{}

	for i, field := range fields {
		if field == pkField {
			val := values[i]
			if str, ok := val.(string); ok && str == "" {
				continue
			}
		}
		filteredFields = append(filteredFields, field)
		filteredValues = append(filteredValues, values[i])
	}

	if len(filteredFields) == 0 {
		return nil, database.NewDBError(database.CodeDBInternal, "no fields to insert").
			WithDetail("table", model.TableName())
	}

	placeholders := make([]string, len(filteredFields))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
		model.TableName(),
		strings.Join(filteredFields, ", "),
		strings.Join(placeholders, ", "),
		pkField,
	)

	nargs := normalizeArgs(filteredValues)
	c.logger.Debug("Create",
		logger.String("query", query),
		logger.String("table", model.TableName()),
	)

	var returnedID interface{}
	if err := c.db.QueryRowContext(ctx, query, nargs...).Scan(&returnedID); err != nil {
		c.logDatabaseError("Create", query, nargs, err)
		return nil, wrapDatabaseError(err, "Create", model.TableName(), query)
	}

	if err := setPrimaryKeyValue(model, pkField, returnedID); err != nil {
		return nil, database.WrapDBError(err, database.CodeDBInternal, "failed to set primary key value").
			WithDetail("table", model.TableName()).
			WithDetail("pk_field", pkField)
	}
	formattedID := formatPrimaryKey(returnedID)
	return &formattedID, nil
}

func (c *client) FindByID(ctx context.Context, model database.Model, id interface{}) *database.DBError {
	fields := getFields(model)
	if len(fields) == 0 {
		return database.NewDBError(database.CodeDBInternal, "no db tags found in model").
			WithDetail("table", model.TableName())
	}

	pkField := getPrimaryKeyField(model)
	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = $1",
		strings.Join(fields, ", "),
		model.TableName(),
		pkField,
	)

	c.logger.Debug("FindByID",
		logger.String("query", query),
		logger.String("table", model.TableName()),
	)

	row := c.db.QueryRowContext(ctx, query, id)
	if err := scanStruct(row, model); err != nil {
		c.logDatabaseError("FindByID", query, []interface{}{id}, err)
		return wrapDatabaseError(err, "FindByID", model.TableName(), query)
	}
	return nil
}

func (c *client) Update(ctx context.Context, model database.Model) *database.DBError {
	fields, values := getFieldsAndValues(model)
	if len(fields) == 0 {
		return database.NewDBError(database.CodeDBInternal, "no db tags found in model").
			WithDetail("table", model.TableName())
	}

	pkField := getPrimaryKeyField(model)
	setParts := make([]string, 0, len(fields))
	updateValues := make([]interface{}, 0, len(values))

	for i, field := range fields {
		if field == pkField || field == "created_at" {
			continue
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, len(setParts)+1))
		updateValues = append(updateValues, values[i])
	}

	if len(setParts) == 0 {
		return database.NewDBError(database.CodeDBInternal, "no fields to update").
			WithDetail("table", model.TableName())
	}

	updateValues = append(updateValues, model.PrimaryKey())

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d",
		model.TableName(),
		strings.Join(setParts, ", "),
		pkField,
		len(updateValues),
	)

	nargs := normalizeArgs(updateValues)
	c.logger.Debug("Update",
		logger.String("query", query),
		logger.String("table", model.TableName()),
		logger.Any("primary_key", model.PrimaryKey()),
	)

	result, err := c.db.ExecContext(ctx, query, nargs...)
	if err != nil {
		c.logDatabaseError("Update", query, nargs, err)
		return wrapDatabaseError(err, "Update", model.TableName(), query)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to get rows affected").
			WithDetail("table", model.TableName())
	}
	if rows == 0 {
		return database.NewDBError(database.CodeDBInternal, "record not found").
			WithDetail("operation", "Update").
			WithDetail("table", model.TableName()).
			WithDetail("primary_key", model.PrimaryKey())
	}

	return nil
}

func (c *client) Delete(ctx context.Context, model database.Model) *database.DBError {
	pkField := getPrimaryKeyField(model)
	query := fmt.Sprintf(
		"UPDATE %s SET deleted_at = $1 WHERE %s = $2 AND deleted_at IS NULL",
		model.TableName(),
		pkField,
	)

	c.logger.Debug("Delete",
		logger.String("query", query),
		logger.String("table", model.TableName()),
	)

	result, err := c.db.ExecContext(ctx, query, time.Now(), model.PrimaryKey())
	if err != nil {
		c.logDatabaseError("Delete", query, []interface{}{time.Now(), model.PrimaryKey()}, err)
		return wrapDatabaseError(err, "Delete", model.TableName(), query)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to get rows affected").
			WithDetail("table", model.TableName())
	}
	if rows == 0 {
		return database.NewDBError(database.CodeDBInternal, "record not found or already deleted").
			WithDetail("operation", "Delete").
			WithDetail("table", model.TableName()).
			WithDetail("primary_key", model.PrimaryKey())
	}

	return nil
}

func (c *client) HardDelete(ctx context.Context, model database.Model) *database.DBError {
	pkField := getPrimaryKeyField(model)
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s = $1",
		model.TableName(),
		pkField,
	)

	c.logger.Debug("HardDelete",
		logger.String("query", query),
		logger.String("table", model.TableName()),
	)

	result, err := c.db.ExecContext(ctx, query, model.PrimaryKey())
	if err != nil {
		c.logDatabaseError("HardDelete", query, []interface{}{model.PrimaryKey()}, err)
		return wrapDatabaseError(err, "HardDelete", model.TableName(), query)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to get rows affected").
			WithDetail("table", model.TableName())
	}
	if rows == 0 {
		return database.NewDBError(database.CodeDBInternal, "record not found").
			WithDetail("operation", "HardDelete").
			WithDetail("table", model.TableName()).
			WithDetail("primary_key", model.PrimaryKey())
	}

	return nil
}

func (c *client) FindOne(ctx context.Context, model database.Model, query string, args ...interface{}) *database.DBError {
	nargs := normalizeArgs(args)
	c.logger.Debug("FindOne", logger.String("query", query))

	row := c.db.QueryRowContext(ctx, query, nargs...)
	if err := scanStruct(row, model); err != nil {
		c.logDatabaseError("FindOne", query, nargs, err)
		return wrapDatabaseError(err, "FindOne", model.TableName(), query)
	}
	return nil
}

func (c *client) FindOneAndUpdate(ctx context.Context, dest interface{}, query string, args ...interface{}) *database.DBError {
	nargs := normalizeArgs(args)
	c.logger.Debug("FindOneAndUpdate", logger.String("query", query))

	row := c.db.QueryRowContext(ctx, query, nargs...)
	if err := scanStruct(row, dest); err != nil {
		c.logDatabaseError("FindOneAndUpdate", query, nargs, err)
		return wrapDatabaseError(err, "FindOneAndUpdate", "Table", query)
	}
	return nil
}

func (c *client) FindMany(ctx context.Context, dest interface{}, query string, args ...interface{}) *database.DBError {
	nargs := normalizeArgs(args)
	c.logger.Debug("FindMany", logger.String("query", query))

	rows, err := c.db.QueryContext(ctx, query, nargs...)
	if err != nil {
		c.logDatabaseError("FindMany", query, nargs, err)
		return wrapDatabaseError(err, "FindMany", "", query)
	}
	defer rows.Close()

	if err := scanStructs(rows, dest, c.logger); err != nil {
		c.logDatabaseError("FindMany:Scan", query, nargs, err)
		return database.WrapDBError(err, database.CodeDBInternal, "failed to scan results").
			WithDetail("operation", "FindMany")
	}
	return nil
}

func (c *client) Exists(ctx context.Context, model database.Model, query string, args ...interface{}) (bool, error) {
	nargs := normalizeArgs(args)
	c.logger.Debug("Exists", logger.String("query", query))

	var exists bool
	err := c.db.QueryRowContext(ctx, query, nargs...).Scan(&exists)
	if err != nil {
		c.logDatabaseError("Exists", query, nargs, err)
		return false, wrapDatabaseError(err, "Exists", model.TableName(), query)
	}
	return exists, nil
}

func (c *client) Count(ctx context.Context, model database.Model, query string, args ...interface{}) (int64, error) {
	nargs := normalizeArgs(args)
	c.logger.Debug("Count", logger.String("query", query))

	var count int64
	err := c.db.QueryRowContext(ctx, query, nargs...).Scan(&count)
	if err != nil {
		c.logDatabaseError("Count", query, nargs, err)
		return 0, wrapDatabaseError(err, "Count", model.TableName(), query)
	}
	return count, nil
}

func (c *client) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	nargs := normalizeArgs(args)
	c.logger.Debug("Query", logger.String("query", query))

	rows, err := c.db.QueryContext(ctx, query, nargs...)
	if err != nil {
		c.logDatabaseError("Query", query, nargs, err)
		return nil, wrapDatabaseError(err, "Query", "", query)
	}
	return &rowsWrapper{rows: rows, log: c.logger}, nil
}

func (c *client) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	nargs := normalizeArgs(args)
	c.logger.Debug("QueryRow", logger.String("query", query))
	return &rowWrapper{row: c.db.QueryRowContext(ctx, query, nargs...), log: c.logger}
}

func (c *client) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	nargs := normalizeArgs(args)
	c.logger.Debug("Exec", logger.String("query", query))

	result, err := c.db.ExecContext(ctx, query, nargs...)
	if err != nil {
		c.logDatabaseError("Exec", query, nargs, err)
		return nil, wrapDatabaseError(err, "Exec", "", query)
	}
	return &resultWrapper{result: result}, nil
}

func (c *client) Begin(ctx context.Context) (database.Transaction, error) {
	c.logger.Debug("Begin transaction")
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		c.logger.Error("Failed to begin transaction", logger.Error(err))
		return nil, database.WrapDBError(err, database.CodeDBInternal, "failed to begin transaction")

	}
	return &transactionWrapper{tx: tx, logger: c.logger}, nil
}

func (c *client) BeginTx(ctx context.Context, opts *database.TxOptions) (database.Transaction, error) {
	c.logger.Debug("Begin transaction with options")
	sqlOpts := &sql.TxOptions{}
	if opts != nil {
		sqlOpts.Isolation = opts.Isolation
		sqlOpts.ReadOnly = opts.ReadOnly
	}

	tx, err := c.db.BeginTx(ctx, sqlOpts)
	if err != nil {
		c.logger.Error("Failed to begin transaction with options", logger.Error(err))
		return nil, database.WrapDBError(err, database.CodeDBInternal, "failed to begin transaction with options")

	}
	return &transactionWrapper{tx: tx, logger: c.logger}, nil
}

func (c *client) WithTransaction(ctx context.Context, fn func(tx database.Transaction) error) error {
	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			c.logger.Error("Transaction panic", logger.Any("panic", p))
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			c.logger.Error("Failed to rollback transaction", logger.Error(rbErr))
			return database.WrapDBError(rbErr, database.CodeDBInternal, "failed to rollback transaction")

		}
		c.logger.Debug("Transaction rolled back", logger.Error(err))
		return err
	}

	if err := tx.Commit(); err != nil {
		c.logger.Error("Failed to commit transaction", logger.Error(err))
		return database.WrapDBError(err, database.CodeDBInternal, "failed to commit transaction")
	}

	c.logger.Debug("Transaction committed successfully")
	return nil
}

func (c *client) Close() error {
	c.logger.Debug("Closing database")
	if err := c.db.Close(); err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to close database")

	}
	return nil
}

func (c *client) Ping(ctx context.Context) *database.DBError {
	if err := c.db.PingContext(ctx); err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to ping database")

	}
	return nil
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
	tx     *sql.Tx
	logger logger.Logger
}

func (c *client) logDatabaseError(operation string, query string, args []interface{}, err error) {
	if err == nil {
		return
	}

	fields := []logger.Field{
		logger.String("operation", operation),
		logger.String("query", query),
		logger.Error(err),
	}

	// Extract PostgreSQL specific error details
	if pqErr, ok := err.(*pq.Error); ok {
		fields = append(fields,
			logger.String("pg_error_code", string(pqErr.Code)),
			logger.String("pg_error_name", pqErr.Code.Name()),
			logger.String("pg_severity", pqErr.Severity),
			logger.String("pg_message", pqErr.Message),
			logger.String("pg_detail", pqErr.Detail),
			logger.String("pg_hint", pqErr.Hint),
			logger.String("pg_position", pqErr.Position),
			logger.String("pg_internal_position", pqErr.InternalPosition),
			logger.String("pg_internal_query", pqErr.InternalQuery),
			logger.String("pg_where", pqErr.Where),
			logger.String("pg_schema", pqErr.Schema),
			logger.String("pg_table", pqErr.Table),
			logger.String("pg_column", pqErr.Column),
			logger.String("pg_datatype", pqErr.DataTypeName),
			logger.String("pg_constraint", pqErr.Constraint),
			logger.String("pg_file", pqErr.File),
			logger.String("pg_line", pqErr.Line),
			logger.String("pg_routine", pqErr.Routine),
		)

		// Detect if error came from a trigger
		if strings.Contains(pqErr.Where, "PL/pgSQL function") ||
			strings.Contains(pqErr.Routine, "trigger") ||
			strings.Contains(pqErr.Message, "trigger") {
			fields = append(fields, logger.String("error_source", "trigger"))
		}

		// Add constraint violation details
		if pqErr.Code.Class() == "23" { // Integrity Constraint Violation
			fields = append(fields, logger.String("constraint_type", "integrity_violation"))
		}
	}

	// Log query arguments (sanitized)
	if len(args) > 0 {
		sanitizedArgs := make([]string, len(args))
		for i, arg := range args {
			sanitizedArgs[i] = fmt.Sprintf("%T", arg)
		}
		fields = append(fields, logger.String("arg_types", strings.Join(sanitizedArgs, ", ")))
	}

	c.logger.Error("Database operation failed", fields...)
}

func (t *transactionWrapper) logDatabaseError(operation string, query string, args []interface{}, err error) {
	if err == nil {
		return
	}

	fields := []logger.Field{
		logger.String("operation", "TX:"+operation),
		logger.String("query", query),
		logger.Error(err),
	}

	if pqErr, ok := err.(*pq.Error); ok {
		fields = append(fields,
			logger.String("pg_error_code", string(pqErr.Code)),
			logger.String("pg_error_name", pqErr.Code.Name()),
			logger.String("pg_severity", pqErr.Severity),
			logger.String("pg_message", pqErr.Message),
			logger.String("pg_detail", pqErr.Detail),
			logger.String("pg_hint", pqErr.Hint),
			logger.String("pg_where", pqErr.Where),
			logger.String("pg_table", pqErr.Table),
			logger.String("pg_column", pqErr.Column),
			logger.String("pg_constraint", pqErr.Constraint),
			logger.String("pg_routine", pqErr.Routine),
		)

		if strings.Contains(pqErr.Where, "PL/pgSQL function") ||
			strings.Contains(pqErr.Routine, "trigger") {
			fields = append(fields, logger.String("error_source", "trigger"))
		}
	}

	t.logger.Error("Transaction operation failed", fields...)
}

func (t *transactionWrapper) Create(ctx context.Context, model database.Model) *database.DBError {
	fields, values := getFieldsAndValues(model)
	if len(fields) == 0 {
		return database.NewDBError(database.CodeDBInternal, "no db tags found in model").
			WithDetail("table", model.TableName())
	}

	placeholders := make([]string, len(fields))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING %s",
		model.TableName(),
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
		getPrimaryKeyField(model),
	)

	nargs := normalizeArgs(values)
	t.logger.Debug("TX Create",
		logger.String("query", query),
		logger.String("table", model.TableName()),
	)

	pkField := getPrimaryKeyField(model)
	var returnedID interface{}

	err := t.tx.QueryRowContext(ctx, query, nargs...).Scan(&returnedID)
	if err != nil {
		t.logDatabaseError("Create", query, nargs, err)
		return wrapDatabaseError(err, "TX:Create", model.TableName(), query)
	}

	if err := setPrimaryKeyValue(model, pkField, returnedID); err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to set primary key value").
			WithDetail("table", model.TableName()).
			WithDetail("pk_field", pkField)
	}

	return nil
}

func (t *transactionWrapper) FindByID(ctx context.Context, model database.Model, id interface{}) *database.DBError {
	fields := getFields(model)
	if len(fields) == 0 {
		return database.NewDBError(database.CodeDBInternal, "no db tags found in model").
			WithDetail("table", model.TableName())
	}

	pkField := getPrimaryKeyField(model)
	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = $1",
		strings.Join(fields, ", "),
		model.TableName(),
		pkField,
	)

	t.logger.Debug("TX FindByID", logger.String("query", query))

	row := t.tx.QueryRowContext(ctx, query, id)
	if err := scanStruct(row, model); err != nil {
		t.logDatabaseError("FindByID", query, []interface{}{id}, err)
		return wrapDatabaseError(err, "TX:FindByID", model.TableName(), query)
	}
	return nil
}

func (t *transactionWrapper) Update(ctx context.Context, model database.Model) *database.DBError {
	fields, values := getFieldsAndValues(model)
	if len(fields) == 0 {
		return database.NewDBError(database.CodeDBInternal, "no db tags found in model").
			WithDetail("table", model.TableName())
	}

	pkField := getPrimaryKeyField(model)
	setParts := make([]string, 0, len(fields))
	updateValues := make([]interface{}, 0, len(values))

	for i, field := range fields {
		if field == pkField || field == "created_at" {
			continue
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, len(setParts)+1))
		updateValues = append(updateValues, values[i])
	}

	if len(setParts) == 0 {
		return database.NewDBError(database.CodeDBInternal, "no fields to update").
			WithDetail("table", model.TableName())
	}

	updateValues = append(updateValues, model.PrimaryKey())

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d",
		model.TableName(),
		strings.Join(setParts, ", "),
		pkField,
		len(updateValues),
	)

	nargs := normalizeArgs(updateValues)
	t.logger.Debug("TX Update",
		logger.String("query", query),
		logger.String("table", model.TableName()),
	)

	result, err := t.tx.ExecContext(ctx, query, nargs...)
	if err != nil {
		t.logDatabaseError("Update", query, nargs, err)
		return wrapDatabaseError(err, "TX:Update", model.TableName(), query)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to get rows affected").
			WithDetail("table", model.TableName())
	}
	if rows == 0 {
		return database.NewDBError(database.CodeDBInternal, "record not found").
			WithDetail("operation", "TX:Update").
			WithDetail("table", model.TableName()).
			WithDetail("primary_key", model.PrimaryKey())
	}

	return nil
}

func (t *transactionWrapper) Delete(ctx context.Context, model database.Model) *database.DBError {
	pkField := getPrimaryKeyField(model)
	query := fmt.Sprintf(
		"UPDATE %s SET deleted_at = $1 WHERE %s = $2 AND deleted_at IS NULL",
		model.TableName(),
		pkField,
	)

	t.logger.Debug("TX Delete", logger.String("query", query))

	result, err := t.tx.ExecContext(ctx, query, time.Now(), model.PrimaryKey())
	if err != nil {
		t.logDatabaseError("Delete", query, []interface{}{time.Now(), model.PrimaryKey()}, err)
		return wrapDatabaseError(err, "TX:Delete", model.TableName(), query)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to get rows affected").
			WithDetail("table", model.TableName())
	}
	if rows == 0 {
		return database.NewDBError(database.CodeDBInternal, "record not found or already deleted").
			WithDetail("operation", "TX:Delete").
			WithDetail("table", model.TableName()).
			WithDetail("primary_key", model.PrimaryKey())
	}

	return nil
}

func (t *transactionWrapper) HardDelete(ctx context.Context, model database.Model) *database.DBError {
	pkField := getPrimaryKeyField(model)
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s = $1",
		model.TableName(),
		pkField,
	)

	t.logger.Debug("TX HardDelete", logger.String("query", query))

	result, err := t.tx.ExecContext(ctx, query, model.PrimaryKey())
	if err != nil {
		t.logDatabaseError("HardDelete", query, []interface{}{model.PrimaryKey()}, err)
		return wrapDatabaseError(err, "TX:HardDelete", model.TableName(), query)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to get rows affected").
			WithDetail("table", model.TableName())
	}
	if rows == 0 {
		return database.NewDBError(database.CodeDBInternal, "record not found").
			WithDetail("operation", "TX:HardDelete").
			WithDetail("table", model.TableName()).
			WithDetail("primary_key", model.PrimaryKey())
	}

	return nil
}

func (t *transactionWrapper) FindOne(ctx context.Context, model database.Model, query string, args ...interface{}) *database.DBError {
	nargs := normalizeArgs(args)
	t.logger.Debug("TX FindOne", logger.String("query", query))

	row := t.tx.QueryRowContext(ctx, query, nargs...)
	if err := scanStruct(row, model); err != nil {
		t.logDatabaseError("FindOne", query, nargs, err)
		return wrapDatabaseError(err, "TX:FindOne", model.TableName(), query)
	}
	return nil
}

func (t *transactionWrapper) FindMany(ctx context.Context, dest interface{}, query string, args ...interface{}) *database.DBError {
	nargs := normalizeArgs(args)
	t.logger.Debug("TX FindMany", logger.String("query", query))

	rows, err := t.tx.QueryContext(ctx, query, nargs...)
	if err != nil {
		t.logDatabaseError("FindMany", query, nargs, err)
		return wrapDatabaseError(err, "TX:FindMany", "", query)
	}
	defer rows.Close()

	if err := scanStructs(rows, dest, t.logger); err != nil {
		t.logDatabaseError("FindMany:Scan", query, nargs, err)
		return database.WrapDBError(err, database.CodeDBInternal, "failed to scan results").
			WithDetail("operation", "TX:FindMany")
	}
	return nil
}

func (t *transactionWrapper) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	nargs := normalizeArgs(args)
	t.logger.Debug("TX Query", logger.String("query", query))

	rows, err := t.tx.QueryContext(ctx, query, nargs...)
	if err != nil {
		t.logDatabaseError("Query", query, nargs, err)
		return nil, wrapDatabaseError(err, "TX:Query", "", query)
	}
	return &rowsWrapper{rows: rows}, nil
}

func (t *transactionWrapper) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	nargs := normalizeArgs(args)
	t.logger.Debug("TX QueryRow", logger.String("query", query))
	return &rowWrapper{row: t.tx.QueryRowContext(ctx, query, nargs...)}
}

func (t *transactionWrapper) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	nargs := normalizeArgs(args)
	t.logger.Debug("TX Exec", logger.String("query", query))

	result, err := t.tx.ExecContext(ctx, query, nargs...)
	if err != nil {
		t.logDatabaseError("Exec", query, nargs, err)
		return nil, wrapDatabaseError(err, "TX:Exec", "", query)
	}
	return &resultWrapper{result: result}, nil
}

func (t *transactionWrapper) Commit() error {
	err := t.tx.Commit()
	if err != nil {
		t.logger.Error("Failed to commit transaction", logger.Error(err))
		return database.WrapDBError(err, database.CodeDBInternal, "failed to commit transaction")

	}
	t.logger.Debug("Transaction committed")
	return nil
}

func (t *transactionWrapper) Rollback() error {
	err := t.tx.Rollback()
	if err != nil {
		t.logger.Error("Failed to rollback transaction", logger.Error(err))
		return database.WrapDBError(err, database.CodeDBInternal, "failed to rollback transaction")

	}
	t.logger.Debug("Transaction rolled back")
	return nil
}

type rowsWrapper struct {
	rows *sql.Rows
	log  logger.Logger
}

func (r *rowsWrapper) Next() bool {
	r.log.Debug("Advancing to next row")
	return r.rows.Next()
}

func (r *rowsWrapper) Scan(dest ...interface{}) error {
	r.log.Debug("Scanning row", logger.Int("num_fields", len(dest)))
	return r.rows.Scan(dest...)
}

func (r *rowsWrapper) ScanOne(model database.Model) *database.DBError {
	r.log.Debug("Scanning one row into model", logger.String("table", model.TableName()))
	if !r.rows.Next() {
		r.log.Debug("No rows found", logger.String("table", model.TableName()))
		if err := r.rows.Err(); err != nil {
			r.log.Error("Failed to iterate rows", logger.Error(err))
			return database.WrapDBError(err, database.CodeDBInternal, "failed to iterate rows")

		}
		return database.NewDBError(database.CodeDBNoRows, "no rows found")
	}
	r.log.Debug("Row found, scanning into model", logger.String("table", model.TableName()))
	if err := scanStructRows(r.rows, model); err != nil {
		r.log.Error("Failed to scan row", logger.Error(err))
		return database.WrapDBError(err, database.CodeDBInternal, "failed to scan row")

	}
	return nil
}

func (r *rowsWrapper) Close() error {
	return r.rows.Close()
}

func (r *rowsWrapper) Err() error {
	return r.rows.Err()
}

type rowWrapper struct {
	row *sql.Row
	log logger.Logger
}

func (r *rowWrapper) Scan(dest ...interface{}) error {
	r.log.Debug("Scanning single row", logger.Int("num_fields", len(dest)))
	return r.row.Scan(dest...)
}

func (r *rowWrapper) ScanOne(model database.Model) error {
	r.log.Debug("Scanning single row into model",
		logger.String("table", model.TableName()),
		logger.String("operation", "ScanOne"),
		logger.Any("model", model),
	)
	return scanStruct(r.row, model)
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

func normalizeArgs(args []interface{}) []interface{} {
	if len(args) == 0 {
		return args
	}
	out := make([]interface{}, len(args))
	for i, a := range args {
		switch v := a.(type) {
		case []string:
			out[i] = pq.Array(v)
		default:
			out[i] = v
		}
	}
	return out
}

func getFields(dest interface{}) []string {
	v := reflect.ValueOf(dest)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	fields := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if tag := field.Tag.Get("db"); tag != "" && tag != "-" {
			fields = append(fields, tag)
		}
	}
	return fields
}

func getFieldsAndValues(src interface{}) ([]string, []interface{}) {
	v := reflect.ValueOf(src)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, nil
	}
	t := v.Type()
	fields := make([]string, 0, t.NumField())
	values := make([]interface{}, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}

		fieldValue := v.Field(i)

		if fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				fields = append(fields, tag)
				values = append(values, nil)
				continue
			}
			fields = append(fields, tag)
			values = append(values, fieldValue.Elem().Interface())
			continue
		}

		if fieldValue.Kind() == reflect.Slice && fieldValue.Type().Elem().Kind() == reflect.Uint8 {
			fields = append(fields, tag)
			values = append(values, fieldValue.Interface())
			continue
		}

		if fieldValue.Kind() == reflect.Struct && fieldValue.Type().String() == "time.Time" {
			timeVal := fieldValue.Interface().(time.Time)
			if timeVal.IsZero() {
				continue
			}
			fields = append(fields, tag)
			values = append(values, timeVal)
			continue
		}

		if fieldValue.Kind() == reflect.Slice && fieldValue.Type().String() == "pq.StringArray" {
			fields = append(fields, tag)
			values = append(values, fieldValue.Interface())
			continue
		}

		fields = append(fields, tag)
		values = append(values, fieldValue.Interface())
	}
	return fields, values
}

func getPrimaryKeyField(model interface{}) string {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if pk := field.Tag.Get("pk"); pk == "true" {
			return field.Tag.Get("db")
		}
	}

	return "id"
}

func setPrimaryKeyValue(model interface{}, pkField string, value interface{}) error {
	v := reflect.ValueOf(model)
	if v.Kind() != reflect.Ptr {
		return database.NewDBError(database.CodeDBInternal, "model must be a pointer").
			WithDetail("pk_field", pkField)
	}
	v = v.Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get("db") == pkField {
			fieldValue := v.Field(i)
			if !fieldValue.CanSet() {
				return database.NewDBError(database.CodeDBInternal, "cannot set primary key field").
					WithDetail("pk_field", pkField).
					WithDetail("field_name", field.Name)
			}

			val := reflect.ValueOf(value)
			if fieldValue.Type() != val.Type() {
				if val.Type().ConvertibleTo(fieldValue.Type()) {
					val = val.Convert(fieldValue.Type())
				}
			}
			fieldValue.Set(val)
			return nil
		}
	}

	return nil
}

func formatPrimaryKey(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprint(v)
	}
}

func scanStruct(row *sql.Row, dest interface{}) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return database.NewDBError(database.CodeDBInternal, "dest must be a pointer to struct")

	}
	v = v.Elem()
	t := v.Type()
	dests := make([]interface{}, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		if tag != "" && tag != "-" {
			fieldValue := v.Field(i)

			if field.Type.String() == "pq.StringArray" {
				dests = append(dests, pq.Array(fieldValue.Addr().Interface()))
			} else if field.Type.Kind() == reflect.Pointer {
				dests = append(dests, fieldValue.Addr().Interface())
			} else if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Uint8 {
				dests = append(dests, fieldValue.Addr().Interface())
			} else if field.Type.Kind() == reflect.Struct {
				if field.Type.String() == "time.Time" {
					dests = append(dests, fieldValue.Addr().Interface())
				} else {
					return fmt.Errorf("unsupported struct type: %s", field.Type.String())
				}
			} else {
				dests = append(dests, fieldValue.Addr().Interface())
			}
		}
	}

	return row.Scan(dests...)
}

func scanStructRows(rows *sql.Rows, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Struct {
		return database.NewDBError(database.CodeDBInternal, "dest must be a pointer to struct")

	}
	destValue = destValue.Elem()
	destType := destValue.Type()
	dests := make([]interface{}, 0, destType.NumField())

	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		tag := field.Tag.Get("db")
		if tag != "" && tag != "-" {
			fieldValue := destValue.Field(i)

			if field.Type.String() == "pq.StringArray" {
				dests = append(dests, pq.Array(fieldValue.Addr().Interface()))
			} else if field.Type.String() == "json.RawMessage" {
				dests = append(dests, fieldValue.Addr().Interface())
				dests = append(dests, fieldValue.Addr().Interface())
			} else if field.Type.Kind() == reflect.Ptr {
				dests = append(dests, fieldValue.Addr().Interface())
			} else if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Uint8 {
				dests = append(dests, fieldValue.Addr().Interface())
			} else if field.Type.Kind() == reflect.Struct {
				if field.Type.String() == "time.Time" {
					dests = append(dests, fieldValue.Addr().Interface())
				} else {
					return database.NewDBError(database.CodeDBInternal, "unsupported struct type").
						WithDetail("type", field.Type.String())
				}
			} else {
				dests = append(dests, fieldValue.Addr().Interface())
			}
		}
	}

	return rows.Scan(dests...)
}

func scanStructs(rows *sql.Rows, dest interface{}, log logger.Logger) error {
	destValue := reflect.ValueOf(dest)
	log.Debug("Scanning multiple rows into slice", logger.String("dest_type", destValue.Type().String()))
	if destValue.Kind() != reflect.Ptr {
		return database.NewDBError(database.CodeDBInternal, "dest must be a pointer to slice")

	}
	sliceValue := destValue.Elem()
	if sliceValue.Kind() != reflect.Slice {
		return database.NewDBError(database.CodeDBInternal, "dest must be a pointer to slice")

	}
	elemType := sliceValue.Type().Elem()
	isPtr := elemType.Kind() == reflect.Ptr
	if isPtr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return database.NewDBError(database.CodeDBInternal, "slice element must be a struct or pointer to struct")

	}

	for rows.Next() {
		log.Debug("Advancing to next row",
			logger.String("element_type", elemType.String()),
		)
		elemValue := reflect.New(elemType)
		elem := elemValue.Elem()
		dests := make([]interface{}, 0, elemType.NumField())

		for i := 0; i < elemType.NumField(); i++ {
			field := elemType.Field(i)
			if tag := field.Tag.Get("db"); tag != "" && tag != "-" {
				fieldValue := elem.Field(i)

				if field.Type.String() == "pq.StringArray" {
					dests = append(dests, pq.Array(fieldValue.Addr().Interface()))
				} else if field.Type.String() == "*pq.StringArray" {
					dests = append(dests, fieldValue.Interface())
				} else {
					dests = append(dests, fieldValue.Addr().Interface())
				}
			}
		}

		if err := rows.Scan(dests...); err != nil {
			return err
		}
		if isPtr {
			sliceValue.Set(reflect.Append(sliceValue, elemValue))
		} else {
			sliceValue.Set(reflect.Append(sliceValue, elem))
		}
	}
	return rows.Err()
}

func wrapDatabaseError(err error, operation, table, query string) *database.DBError {
	if err == nil {
		return nil
	}

	var dbErr *database.DBError
	if errors.As(err, &dbErr) {
		return dbErr
	}

	if err == sql.ErrNoRows {
		return database.NewDBError(database.CodeDBNoRows, "No rows found").
			WithOperation(operation).
			WithTable(table).
			WithQuery(query).
			WithWrapped(err)
	}

	if err == sql.ErrTxDone {
		return database.NewDBError(database.CodeDBTransaction, "Transaction already committed or rolled back").
			WithOperation(operation).
			WithTable(table).
			WithWrapped(err)
	}

	if err == sql.ErrConnDone {
		return database.NewDBError(database.CodeDBConnection, "Connection already closed").
			WithOperation(operation).
			WithTable(table).
			WithWrapped(err)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return database.NewDBError(database.CodeDBTimeout, "Operation timed out").
			WithOperation(operation).
			WithTable(table).
			WithQuery(query).
			WithWrapped(err)
	}

	if errors.Is(err, context.Canceled) {
		return database.NewDBError(database.CodeDBTimeout, "Operation canceled").
			WithOperation(operation).
			WithTable(table).
			WithQuery(query).
			WithWrapped(err)
	}

	if strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "no such host") ||
		strings.Contains(err.Error(), "network is unreachable") {
		return database.NewDBError(database.CodeDBConnection, "Connection failed").
			WithOperation(operation).
			WithTable(table).
			WithWrapped(err)
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		converted := ConvertPQError(err, operation, query)
		if dbErr, ok := converted.(*database.DBError); ok {
			if table != "" && dbErr.Table() == "" {
				dbErr.WithTable(table)
			}
			return dbErr
		}
	}

	return database.NewDBError(database.CodeDBInternal, "Database operation failed").
		WithOperation(operation).
		WithTable(table).
		WithQuery(query).
		WithWrapped(err)
}
