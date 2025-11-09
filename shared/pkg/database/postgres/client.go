package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"shared/pkg/database"
	apperrors "shared/pkg/errors"
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
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "no db tags found in model").
			WithService("postgres-client").
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
		return nil, apperrors.New(apperrors.CodeInvalidArgument, "no fields to insert").
			WithService("postgres-client").
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
		return nil, apperrors.FromError(err, apperrors.CodeInternal, "failed to set primary key value").
			WithService("postgres-client").
			WithDetail("table", model.TableName()).
			WithDetail("pk_field", pkField)
	}
	formattedID := formatPrimaryKey(returnedID)
	return &formattedID, nil
}

func (c *client) FindByID(ctx context.Context, model database.Model, id interface{}) apperrors.AppError {
	fields := getFields(model)
	if len(fields) == 0 {
		return apperrors.New(apperrors.CodeInvalidArgument, "no db tags found in model").
			WithService("postgres-client").
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

func (c *client) Update(ctx context.Context, model database.Model) apperrors.AppError {
	fields, values := getFieldsAndValues(model)
	if len(fields) == 0 {
		return apperrors.New(apperrors.CodeInvalidArgument, "no db tags found in model").
			WithService("postgres-client").
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
		return apperrors.New(apperrors.CodeInvalidArgument, "no fields to update").
			WithService("postgres-client").
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
		return apperrors.FromError(err, apperrors.CodeInternal, "failed to get rows affected").
			WithService("postgres-client").
			WithDetail("table", model.TableName())
	}
	if rows == 0 {
		return apperrors.New(apperrors.CodeNotFound, "record not found").
			WithService("postgres-client").
			WithDetail("operation", "Update").
			WithDetail("table", model.TableName()).
			WithDetail("primary_key", model.PrimaryKey())
	}

	return nil
}

func (c *client) Delete(ctx context.Context, model database.Model) apperrors.AppError {
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
		return apperrors.FromError(err, apperrors.CodeInternal, "failed to get rows affected").
			WithService("postgres-client").
			WithDetail("table", model.TableName())
	}
	if rows == 0 {
		return apperrors.New(apperrors.CodeNotFound, "record not found or already deleted").
			WithService("postgres-client").
			WithDetail("operation", "Delete").
			WithDetail("table", model.TableName()).
			WithDetail("primary_key", model.PrimaryKey())
	}

	return nil
}

func (c *client) HardDelete(ctx context.Context, model database.Model) apperrors.AppError {
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
		return apperrors.FromError(err, apperrors.CodeInternal, "failed to get rows affected").
			WithService("postgres-client").
			WithDetail("table", model.TableName())
	}
	if rows == 0 {
		return apperrors.New(apperrors.CodeNotFound, "record not found").
			WithService("postgres-client").
			WithDetail("operation", "HardDelete").
			WithDetail("table", model.TableName()).
			WithDetail("primary_key", model.PrimaryKey())
	}

	return nil
}

func (c *client) FindOne(ctx context.Context, model database.Model, query string, args ...interface{}) apperrors.AppError {
	nargs := normalizeArgs(args)
	c.logger.Debug("FindOne", logger.String("query", query))

	row := c.db.QueryRowContext(ctx, query, nargs...)
	if err := scanStruct(row, model); err != nil {
		c.logDatabaseError("FindOne", query, nargs, err)
		return wrapDatabaseError(err, "FindOne", model.TableName(), query)
	}
	return nil
}

func (c *client) FindMany(ctx context.Context, dest interface{}, query string, args ...interface{}) apperrors.AppError {
	nargs := normalizeArgs(args)
	c.logger.Debug("FindMany", logger.String("query", query))

	rows, err := c.db.QueryContext(ctx, query, nargs...)
	if err != nil {
		c.logDatabaseError("FindMany", query, nargs, err)
		return wrapDatabaseError(err, "FindMany", "", query)
	}
	defer rows.Close()

	if err := scanStructs(rows, dest); err != nil {
		c.logDatabaseError("FindMany:Scan", query, nargs, err)
		return apperrors.FromError(err, apperrors.CodeInternal, "failed to scan results").
			WithService("postgres-client").
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
	return &rowsWrapper{rows: rows}, nil
}

func (c *client) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	nargs := normalizeArgs(args)
	c.logger.Debug("QueryRow", logger.String("query", query))
	return &rowWrapper{row: c.db.QueryRowContext(ctx, query, nargs...)}
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
		return nil, apperrors.FromError(err, apperrors.CodeInternal, "failed to begin transaction").
			WithService("postgres-client")
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
		return nil, apperrors.FromError(err, apperrors.CodeInternal, "failed to begin transaction with options").
			WithService("postgres-client")
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
			return apperrors.FromError(rbErr, apperrors.CodeInternal, "failed to rollback transaction").
				WithService("postgres-client")
		}
		c.logger.Debug("Transaction rolled back", logger.Error(err))
		return err
	}

	if err := tx.Commit(); err != nil {
		c.logger.Error("Failed to commit transaction", logger.Error(err))
		return apperrors.FromError(err, apperrors.CodeInternal, "failed to commit transaction").
			WithService("postgres-client")
	}

	c.logger.Debug("Transaction committed successfully")
	return nil
}

func (c *client) Close() error {
	c.logger.Debug("Closing database")
	if err := c.db.Close(); err != nil {
		return apperrors.FromError(err, apperrors.CodeInternal, "failed to close database").
			WithService("postgres-client")
	}
	return nil
}

func (c *client) Ping(ctx context.Context) apperrors.AppError {
	if err := c.db.PingContext(ctx); err != nil {
		return apperrors.FromError(err, apperrors.CodeInternal, "failed to ping database").
			WithService("postgres-client")
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

func (t *transactionWrapper) Create(ctx context.Context, model database.Model) apperrors.AppError {
	fields, values := getFieldsAndValues(model)
	if len(fields) == 0 {
		return apperrors.New(apperrors.CodeInvalidArgument, "no db tags found in model").
			WithService("postgres-client").
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
		return apperrors.FromError(err, apperrors.CodeInternal, "failed to set primary key value").
			WithService("postgres-client").
			WithDetail("table", model.TableName()).
			WithDetail("pk_field", pkField)
	}

	return nil
}

func (t *transactionWrapper) FindByID(ctx context.Context, model database.Model, id interface{}) apperrors.AppError {
	fields := getFields(model)
	if len(fields) == 0 {
		return apperrors.New(apperrors.CodeInvalidArgument, "no db tags found in model").
			WithService("postgres-client").
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

func (t *transactionWrapper) Update(ctx context.Context, model database.Model) apperrors.AppError {
	fields, values := getFieldsAndValues(model)
	if len(fields) == 0 {
		return apperrors.New(apperrors.CodeInvalidArgument, "no db tags found in model").
			WithService("postgres-client").
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
		return apperrors.New(apperrors.CodeInvalidArgument, "no fields to update").
			WithService("postgres-client").
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
		return apperrors.FromError(err, apperrors.CodeInternal, "failed to get rows affected").
			WithService("postgres-client").
			WithDetail("table", model.TableName())
	}
	if rows == 0 {
		return apperrors.New(apperrors.CodeNotFound, "record not found").
			WithService("postgres-client").
			WithDetail("operation", "TX:Update").
			WithDetail("table", model.TableName()).
			WithDetail("primary_key", model.PrimaryKey())
	}

	return nil
}

func (t *transactionWrapper) Delete(ctx context.Context, model database.Model) apperrors.AppError {
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
		return apperrors.FromError(err, apperrors.CodeInternal, "failed to get rows affected").
			WithService("postgres-client").
			WithDetail("table", model.TableName())
	}
	if rows == 0 {
		return apperrors.New(apperrors.CodeNotFound, "record not found or already deleted").
			WithService("postgres-client").
			WithDetail("operation", "TX:Delete").
			WithDetail("table", model.TableName()).
			WithDetail("primary_key", model.PrimaryKey())
	}

	return nil
}

func (t *transactionWrapper) HardDelete(ctx context.Context, model database.Model) apperrors.AppError {
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
		return apperrors.FromError(err, apperrors.CodeInternal, "failed to get rows affected").
			WithService("postgres-client").
			WithDetail("table", model.TableName())
	}
	if rows == 0 {
		return apperrors.New(apperrors.CodeNotFound, "record not found").
			WithService("postgres-client").
			WithDetail("operation", "TX:HardDelete").
			WithDetail("table", model.TableName()).
			WithDetail("primary_key", model.PrimaryKey())
	}

	return nil
}

func (t *transactionWrapper) FindOne(ctx context.Context, model database.Model, query string, args ...interface{}) apperrors.AppError {
	nargs := normalizeArgs(args)
	t.logger.Debug("TX FindOne", logger.String("query", query))

	row := t.tx.QueryRowContext(ctx, query, nargs...)
	if err := scanStruct(row, model); err != nil {
		t.logDatabaseError("FindOne", query, nargs, err)
		return wrapDatabaseError(err, "TX:FindOne", model.TableName(), query)
	}
	return nil
}

func (t *transactionWrapper) FindMany(ctx context.Context, dest interface{}, query string, args ...interface{}) apperrors.AppError {
	nargs := normalizeArgs(args)
	t.logger.Debug("TX FindMany", logger.String("query", query))

	rows, err := t.tx.QueryContext(ctx, query, nargs...)
	if err != nil {
		t.logDatabaseError("FindMany", query, nargs, err)
		return wrapDatabaseError(err, "TX:FindMany", "", query)
	}
	defer rows.Close()

	if err := scanStructs(rows, dest); err != nil {
		t.logDatabaseError("FindMany:Scan", query, nargs, err)
		return apperrors.FromError(err, apperrors.CodeInternal, "failed to scan results").
			WithService("postgres-client").
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
		return apperrors.FromError(err, apperrors.CodeInternal, "failed to commit transaction").
			WithService("postgres-client")
	}
	t.logger.Debug("Transaction committed")
	return nil
}

func (t *transactionWrapper) Rollback() error {
	err := t.tx.Rollback()
	if err != nil {
		t.logger.Error("Failed to rollback transaction", logger.Error(err))
		return apperrors.FromError(err, apperrors.CodeInternal, "failed to rollback transaction").
			WithService("postgres-client")
	}
	t.logger.Debug("Transaction rolled back")
	return nil
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

func (r *rowsWrapper) ScanOne(model database.Model) apperrors.AppError {
	if !r.rows.Next() {
		if err := r.rows.Err(); err != nil {
			return apperrors.FromError(err, apperrors.CodeDatabaseError, "failed to iterate rows").
				WithService("postgres-client")
		}
		return apperrors.FromError(sql.ErrNoRows, apperrors.CodeNotFound, "no rows found").
			WithService("postgres-client")
	}
	if err := scanStructRows(r.rows, model); err != nil {
		return apperrors.FromError(err, apperrors.CodeDatabaseError, "failed to scan row").
			WithService("postgres-client")
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
}

func (r *rowWrapper) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}

func (r *rowWrapper) ScanOne(model database.Model) apperrors.AppError {
	if err := scanStruct(r.row, model); err != nil {
		return wrapDatabaseError(err, "ScanOne", model.TableName(), "")
	}
	return nil
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
			fields = append(fields, tag)
			values = append(values, fieldValue.Interface())
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
		return apperrors.New(apperrors.CodeInvalidArgument, "model must be a pointer").
			WithService("postgres-client").
			WithDetail("pk_field", pkField)
	}
	v = v.Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get("db") == pkField {
			fieldValue := v.Field(i)
			if !fieldValue.CanSet() {
				return apperrors.New(apperrors.CodeInternal, "cannot set primary key field").
					WithService("postgres-client").
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
		return apperrors.New(apperrors.CodeInvalidArgument, "dest must be a pointer to struct").
			WithService("postgres-client")
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
		return apperrors.New(apperrors.CodeInvalidArgument, "dest must be a pointer to struct").
			WithService("postgres-client")
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
			} else if field.Type.Kind() == reflect.Ptr {
				dests = append(dests, fieldValue.Addr().Interface())
			} else if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Uint8 {
				dests = append(dests, fieldValue.Addr().Interface())
			} else if field.Type.Kind() == reflect.Struct {
				if field.Type.String() == "time.Time" {
					dests = append(dests, fieldValue.Addr().Interface())
				} else {
					return apperrors.New(apperrors.CodeInvalidArgument, "unsupported struct type").
						WithService("postgres-client").
						WithDetail("type", field.Type.String())
				}
			} else {
				dests = append(dests, fieldValue.Addr().Interface())
			}
		}
	}

	return rows.Scan(dests...)
}

func scanStructs(rows *sql.Rows, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return apperrors.New(apperrors.CodeInvalidArgument, "dest must be a pointer to slice").
			WithService("postgres-client")
	}
	sliceValue := destValue.Elem()
	if sliceValue.Kind() != reflect.Slice {
		return apperrors.New(apperrors.CodeInvalidArgument, "dest must be a pointer to slice").
			WithService("postgres-client")
	}
	elemType := sliceValue.Type().Elem()
	isPtr := elemType.Kind() == reflect.Ptr
	if isPtr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return apperrors.New(apperrors.CodeInvalidArgument, "slice element must be a struct or pointer to struct").
			WithService("postgres-client")
	}

	for rows.Next() {
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

func wrapDatabaseError(err error, operation, table, query string) apperrors.AppError {
	if err == nil {
		return nil
	}

	if appErr, ok := err.(apperrors.AppError); ok {
		return appErr
	}

	if err == sql.ErrNoRows {
		return apperrors.New(apperrors.CodeNotFound, "record not found").
			WithService("postgres-client").
			WithDetail("operation", operation).
			WithDetail("table", table).
			WithDetail("query", query)
	}

	if err == sql.ErrTxDone {
		return apperrors.New(apperrors.CodeDatabaseError, "transaction already committed or rolled back").
			WithService("postgres-client").
			WithDetail("operation", operation).
			WithDetail("table", table).
			WithDetail("error_type", "transaction_done")
	}

	if err == sql.ErrConnDone {
		return apperrors.New(apperrors.CodeDatabaseError, "connection already closed").
			WithService("postgres-client").
			WithDetail("operation", operation).
			WithDetail("table", table).
			WithDetail("error_type", "connection_closed")
	}

	if pqErr, ok := err.(*pq.Error); ok {
		return handlePostgresError(pqErr, operation, table, query, err)
	}

	if strings.Contains(err.Error(), "context deadline exceeded") {
		return apperrors.FromError(err, apperrors.CodeTimeout, "database operation timeout").
			WithService("postgres-client").
			WithDetail("operation", operation).
			WithDetail("table", table).
			WithDetail("query", query)
	}

	if strings.Contains(err.Error(), "context canceled") {
		return apperrors.FromError(err, apperrors.CodeCancelled, "database operation canceled").
			WithService("postgres-client").
			WithDetail("operation", operation).
			WithDetail("table", table).
			WithDetail("query", query)
	}

	if strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "no such host") ||
		strings.Contains(err.Error(), "network is unreachable") {
		return apperrors.FromError(err, apperrors.CodeUnavailable, "database connection failed").
			WithService("postgres-client").
			WithDetail("operation", operation).
			WithDetail("table", table).
			WithDetail("error_type", "connection_failed")
	}

	return apperrors.FromError(err, apperrors.CodeDatabaseError, "database operation failed").
		WithService("postgres-client").
		WithDetail("operation", operation).
		WithDetail("table", table).
		WithDetail("query", query).
		WithDetail("error_type", "unknown")
}

func handlePostgresError(pqErr *pq.Error, operation, table, query string, originalErr error) apperrors.AppError {
	class := pqErr.Code.Class()
	code := string(pqErr.Code)

	var errCode, errMsg string
	var addDetails func(apperrors.AppError) apperrors.AppError

	switch class {
	case "23":
		errCode, errMsg, addDetails = getIntegrityConstraintViolation(pqErr, code)
	case "22":
		errCode, errMsg, addDetails = getDataException(pqErr, code)
	case "42":
		errCode, errMsg, addDetails = getSyntaxErrorOrAccessRuleViolation(pqErr, code)
	case "08":
		errCode, errMsg, addDetails = getConnectionException(pqErr, code)
	case "40":
		errCode, errMsg, addDetails = getTransactionRollback(pqErr, code)
	case "53":
		errCode, errMsg, addDetails = getInsufficientResources(pqErr, code)
	case "54":
		errCode, errMsg, addDetails = getProgramLimitExceeded(pqErr, code)
	case "57":
		errCode, errMsg, addDetails = getOperatorIntervention(pqErr, code)
	case "58":
		errCode, errMsg, addDetails = getSystemError(pqErr, code)
	default:
		errCode = apperrors.CodeDatabaseError
		errMsg = "database operation failed"
		addDetails = func(e apperrors.AppError) apperrors.AppError {
			return e.WithDetail("pg_error_class", class)
		}
	}

	baseErr := apperrors.New(errCode, errMsg).
		WithService("postgres-client").
		WithDetail("operation", operation).
		WithDetail("table", table).
		WithDetail("query", query).
		WithDetail("pg_error_code", code).
		WithDetail("pg_error_name", pqErr.Code.Name()).
		WithDetail("pg_severity", pqErr.Severity).
		WithDetail("pg_message", pqErr.Message).
		WithDetail("wrapped_error", originalErr.Error())

	if pqErr.Detail != "" {
		baseErr = baseErr.WithDetail("pg_detail", pqErr.Detail)
	}
	if pqErr.Hint != "" {
		baseErr = baseErr.WithDetail("pg_hint", pqErr.Hint)
	}
	if pqErr.Constraint != "" {
		baseErr = baseErr.WithDetail("pg_constraint", pqErr.Constraint)
	}
	if pqErr.Table != "" {
		baseErr = baseErr.WithDetail("pg_table", pqErr.Table)
	}
	if pqErr.Column != "" {
		baseErr = baseErr.WithDetail("pg_column", pqErr.Column)
	}
	if pqErr.Schema != "" {
		baseErr = baseErr.WithDetail("pg_schema", pqErr.Schema)
	}
	if pqErr.DataTypeName != "" {
		baseErr = baseErr.WithDetail("pg_datatype", pqErr.DataTypeName)
	}
	if pqErr.Where != "" {
		baseErr = baseErr.WithDetail("pg_where", pqErr.Where)
	}

	if strings.Contains(pqErr.Where, "PL/pgSQL function") ||
		strings.Contains(pqErr.Routine, "trigger") ||
		strings.Contains(pqErr.Message, "trigger") {
		baseErr = baseErr.WithDetail("error_source", "trigger")
	}

	if addDetails != nil {
		baseErr = addDetails(baseErr)
	}

	return baseErr
}

func getIntegrityConstraintViolation(pqErr *pq.Error, code string) (string, string, func(apperrors.AppError) apperrors.AppError) {
	switch code {
	case "23505":
		return apperrors.CodeAlreadyExists, "record already exists", func(e apperrors.AppError) apperrors.AppError {
			return e.WithDetail("constraint", pqErr.Constraint)
		}
	case "23503":
		return apperrors.CodeInvalidArgument, "foreign key constraint violation", func(e apperrors.AppError) apperrors.AppError {
			return e.WithDetail("constraint", pqErr.Constraint)
		}
	case "23502":
		return apperrors.CodeInvalidArgument, "not null constraint violation", func(e apperrors.AppError) apperrors.AppError {
			return e.WithDetail("column", pqErr.Column)
		}
	case "23514":
		return apperrors.CodeInvalidArgument, "check constraint violation", func(e apperrors.AppError) apperrors.AppError {
			return e.WithDetail("constraint", pqErr.Constraint)
		}
	case "23001":
		return apperrors.CodeInvalidArgument, "restrict violation", nil
	default:
		return apperrors.CodeDatabaseError, "integrity constraint violation", func(e apperrors.AppError) apperrors.AppError {
			return e.WithDetail("constraint_type", "integrity_violation")
		}
	}
}

func getDataException(pqErr *pq.Error, code string) (string, string, func(apperrors.AppError) apperrors.AppError) {
	var errorType string
	switch code {
	case "22001":
		errorType = "string_data_right_truncation"
	case "22003":
		errorType = "numeric_value_out_of_range"
	case "22007":
		errorType = "invalid_datetime_format"
	case "22008":
		errorType = "datetime_field_overflow"
	case "22012":
		errorType = "division_by_zero"
	case "22P02":
		errorType = "invalid_text_representation"
	case "22P03":
		errorType = "invalid_binary_representation"
	case "22P04":
		errorType = "bad_copy_file_format"
	default:
		errorType = "data_exception"
	}

	return apperrors.CodeInvalidArgument, "invalid data", func(e apperrors.AppError) apperrors.AppError {
		e = e.WithDetail("error_type", errorType)
		if pqErr.Column != "" {
			e = e.WithDetail("column", pqErr.Column)
		}
		return e
	}
}

func getSyntaxErrorOrAccessRuleViolation(pqErr *pq.Error, code string) (string, string, func(apperrors.AppError) apperrors.AppError) {
	switch code {
	case "42501":
		return apperrors.CodePermissionDenied, "insufficient privilege", nil
	case "42601":
		return apperrors.CodeInternal, "syntax error", func(e apperrors.AppError) apperrors.AppError {
			return e.WithDetail("error_type", "syntax_error")
		}
	case "42703":
		return apperrors.CodeInvalidArgument, "undefined column", func(e apperrors.AppError) apperrors.AppError {
			return e.WithDetail("column", pqErr.Column)
		}
	case "42P01":
		return apperrors.CodeNotFound, "undefined table", nil
	case "42P02":
		return apperrors.CodeInternal, "undefined parameter", nil
	case "42883":
		return apperrors.CodeInternal, "undefined function", nil
	default:
		return apperrors.CodeInternal, "syntax error or access rule violation", nil
	}
}

func getConnectionException(pqErr *pq.Error, code string) (string, string, func(apperrors.AppError) apperrors.AppError) {
	var errorType string
	switch code {
	case "08000":
		errorType = "connection_exception"
	case "08003":
		errorType = "connection_does_not_exist"
	case "08006":
		errorType = "connection_failure"
	case "08001":
		errorType = "sqlclient_unable_to_establish_connection"
	case "08004":
		errorType = "sqlserver_rejected_connection"
	case "08007":
		errorType = "transaction_resolution_unknown"
	case "08P01":
		errorType = "protocol_violation"
	default:
		errorType = "connection_exception"
	}

	return apperrors.CodeUnavailable, "connection exception", func(e apperrors.AppError) apperrors.AppError {
		return e.WithDetail("error_type", errorType)
	}
}

func getTransactionRollback(pqErr *pq.Error, code string) (string, string, func(apperrors.AppError) apperrors.AppError) {
	switch code {
	case "40P01":
		return apperrors.CodeDeadlock, "deadlock detected", nil
	case "40001":
		return apperrors.CodeAborted, "transaction rollback", func(e apperrors.AppError) apperrors.AppError {
			return e.WithDetail("error_type", "serialization_failure")
		}
	case "40002":
		return apperrors.CodeAborted, "transaction rollback", func(e apperrors.AppError) apperrors.AppError {
			return e.WithDetail("error_type", "transaction_integrity_constraint_violation")
		}
	case "40003":
		return apperrors.CodeAborted, "transaction rollback", func(e apperrors.AppError) apperrors.AppError {
			return e.WithDetail("error_type", "statement_completion_unknown")
		}
	default:
		return apperrors.CodeAborted, "transaction rollback", nil
	}
}

func getInsufficientResources(pqErr *pq.Error, code string) (string, string, func(apperrors.AppError) apperrors.AppError) {
	var errorType string
	switch code {
	case "53000":
		errorType = "insufficient_resources"
	case "53100":
		errorType = "disk_full"
	case "53200":
		errorType = "out_of_memory"
	case "53300":
		errorType = "too_many_connections"
	case "53400":
		errorType = "configuration_limit_exceeded"
	default:
		errorType = "insufficient_resources"
	}

	return apperrors.CodeResourceExhausted, "insufficient resources", func(e apperrors.AppError) apperrors.AppError {
		return e.WithDetail("error_type", errorType)
	}
}

func getProgramLimitExceeded(pqErr *pq.Error, code string) (string, string, func(apperrors.AppError) apperrors.AppError) {
	var errorType string
	switch code {
	case "54000":
		errorType = "program_limit_exceeded"
	case "54001":
		errorType = "statement_too_complex"
	case "54011":
		errorType = "too_many_columns"
	case "54023":
		errorType = "too_many_arguments"
	default:
		errorType = "program_limit_exceeded"
	}

	return apperrors.CodeResourceExhausted, "program limit exceeded", func(e apperrors.AppError) apperrors.AppError {
		return e.WithDetail("error_type", errorType)
	}
}

func getOperatorIntervention(pqErr *pq.Error, code string) (string, string, func(apperrors.AppError) apperrors.AppError) {
	switch code {
	case "57014":
		return apperrors.CodeCancelled, "query canceled", nil
	case "57000", "57P01", "57P02", "57P03", "57P04":
		var errorType string
		switch code {
		case "57000":
			errorType = "operator_intervention"
		case "57P01":
			errorType = "admin_shutdown"
		case "57P02":
			errorType = "crash_shutdown"
		case "57P03":
			errorType = "cannot_connect_now"
		case "57P04":
			errorType = "database_dropped"
		}
		return apperrors.CodeUnavailable, "operator intervention", func(e apperrors.AppError) apperrors.AppError {
			return e.WithDetail("error_type", errorType)
		}
	default:
		return apperrors.CodeUnavailable, "operator intervention", nil
	}
}

func getSystemError(pqErr *pq.Error, code string) (string, string, func(apperrors.AppError) apperrors.AppError) {
	var errorType string
	switch code {
	case "58000":
		errorType = "system_error"
	case "58030":
		errorType = "io_error"
	case "58P01":
		errorType = "undefined_file"
	case "58P02":
		errorType = "duplicate_file"
	default:
		errorType = "system_error"
	}

	return apperrors.CodeInternal, "system error", func(e apperrors.AppError) apperrors.AppError {
		return e.WithDetail("error_type", errorType).
			WithDetail("pg_errno", pqErr.Code)
	}
}
