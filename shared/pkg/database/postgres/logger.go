package postgres

import (
	"fmt"
	"strings"

	"shared/pkg/database"
	"shared/pkg/logger"

	"github.com/lib/pq"
)

// dbLogger provides structured logging for database operations.
type dbLogger struct {
	logger.Logger
}

// newDBLogger creates a new database logger wrapper.
func newDBLogger(l logger.Logger) *dbLogger {
	return &dbLogger{Logger: l}
}

// ConnectionError logs a connection-related error.
func (l *dbLogger) ConnectionError(action string, err error) {
	l.Error("Database connection failed",
		logger.String("action", action),
		logger.Error(err),
	)
}

// QueryStart logs the beginning of a database operation.
func (l *dbLogger) QueryStart(operation, table string, fieldCount int) {
	fields := []logger.Field{
		logger.String("operation", operation),
		logger.String("table", table),
	}
	if fieldCount > 0 {
		fields = append(fields, logger.Int("fields", fieldCount))
	}
	l.Info("Executing query", fields...)
}

// QuerySuccess logs successful completion of a database operation.
func (l *dbLogger) QuerySuccess(operation, table, id string) {
	fields := []logger.Field{
		logger.String("operation", operation),
		logger.String("table", table),
	}
	if id != "" {
		fields = append(fields, logger.String("id", id))
	}
	l.Info("Query completed", fields...)
}

// QueryError logs a database operation error with full context.
func (l *dbLogger) QueryError(operation, table, query string, args []interface{}, err error) *database.DBError {
	fields := buildErrorLogFields(operation, table, query, args, err)
	l.Error("Query failed", fields...)
	return wrapDatabaseError(err, operation, table, query)
}

// TxQueryStart logs the beginning of a transaction operation.
func (l *dbLogger) TxQueryStart(operation, table string, fieldCount int) {
	fields := []logger.Field{
		logger.String("operation", "TX:"+operation),
		logger.String("table", table),
	}
	if fieldCount > 0 {
		fields = append(fields, logger.Int("fields", fieldCount))
	}
	l.Info("Executing transaction query", fields...)
}

// TxQuerySuccess logs successful completion of a transaction operation.
func (l *dbLogger) TxQuerySuccess(operation, table, id string) {
	fields := []logger.Field{
		logger.String("operation", "TX:"+operation),
		logger.String("table", table),
	}
	if id != "" {
		fields = append(fields, logger.String("id", id))
	}
	l.Info("Transaction query completed", fields...)
}

// TxQueryError logs a transaction operation error with full context.
func (l *dbLogger) TxQueryError(operation, table, query string, args []interface{}, err error) *database.DBError {
	fields := buildErrorLogFields("TX:"+operation, table, query, args, err)
	l.Error("Transaction query failed", fields...)
	return wrapDatabaseError(err, "TX:"+operation, table, query)
}

// NoFieldsError creates and logs an error for models with no insertable fields.
func (l *dbLogger) NoFieldsError(operation, table string) *database.DBError {
	l.Error("No database fields found",
		logger.String("operation", operation),
		logger.String("table", table),
	)
	return database.NewDBError(database.CodeDBInternal, "no db tags found in model").
		WithDetail("table", table)
}

// buildErrorLogFields creates log fields for error logging.
func buildErrorLogFields(operation, table, query string, args []interface{}, err error) []logger.Field {
	fields := []logger.Field{
		logger.String("operation", operation),
		logger.Error(err),
	}

	if table != "" {
		fields = append(fields, logger.String("table", table))
	}

	if query != "" {
		fields = append(fields, logger.String("query", truncateForLog(query, 500)))
	}

	// Add argument types for debugging (not values for security)
	if len(args) > 0 {
		fields = append(fields, logger.String("arg_types", formatArgTypesForLog(args)))
	}

	// Add PostgreSQL-specific error details
	if pqErr := extractPQError(err); pqErr != nil {
		fields = append(fields, buildPQErrorFields(pqErr)...)
	}

	return fields
}

// extractPQError attempts to extract a pq.Error from an error chain.
func extractPQError(err error) *pq.Error {
	var pqErr *pq.Error
	if e, ok := err.(*pq.Error); ok {
		pqErr = e
	}
	return pqErr
}

// buildPQErrorFields creates log fields from a PostgreSQL error.
func buildPQErrorFields(pqErr *pq.Error) []logger.Field {
	fields := []logger.Field{
		logger.String("pg_code", string(pqErr.Code)),
		logger.String("pg_severity", pqErr.Severity),
		logger.String("pg_message", pqErr.Message),
	}

	// Add optional fields only if they have values
	optionalFields := map[string]string{
		"pg_detail":     pqErr.Detail,
		"pg_hint":       pqErr.Hint,
		"pg_table":      pqErr.Table,
		"pg_column":     pqErr.Column,
		"pg_constraint": pqErr.Constraint,
		"pg_schema":     pqErr.Schema,
		"pg_where":      pqErr.Where,
	}

	for key, value := range optionalFields {
		if value != "" {
			fields = append(fields, logger.String(key, value))
		}
	}

	// Detect trigger-related errors
	if isTriggerError(pqErr) {
		fields = append(fields, logger.String("error_source", "trigger"))
	}

	// Detect constraint violations
	if pqErr.Code.Class() == "23" {
		fields = append(fields, logger.String("constraint_type", "integrity_violation"))
	}

	return fields
}

// isTriggerError checks if a PostgreSQL error originated from a trigger.
func isTriggerError(pqErr *pq.Error) bool {
	return strings.Contains(pqErr.Where, "PL/pgSQL function") ||
		strings.Contains(pqErr.Routine, "trigger") ||
		strings.Contains(pqErr.Message, "trigger")
}

// formatArgTypesForLog formats argument types for logging (not values for security).
func formatArgTypesForLog(args []interface{}) string {
	if len(args) == 0 {
		return ""
	}

	types := make([]string, len(args))
	for i, arg := range args {
		types[i] = fmt.Sprintf("%T", arg)
	}
	return strings.Join(types, ", ")
}

// truncateForLog truncates a string for log output.
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// LogLevel represents logging verbosity levels.
type LogLevel int

const (
	LogLevelError LogLevel = iota
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// OperationContext holds context for a database operation for logging purposes.
type OperationContext struct {
	Operation  string
	Table      string
	PrimaryKey interface{}
	Query      string
	FieldCount int
}

// LogOperation provides a structured way to log database operations.
func (l *dbLogger) LogOperation(ctx OperationContext, level LogLevel, message string, extraFields ...logger.Field) {
	fields := []logger.Field{
		logger.String("operation", ctx.Operation),
	}

	if ctx.Table != "" {
		fields = append(fields, logger.String("table", ctx.Table))
	}
	if ctx.PrimaryKey != nil {
		fields = append(fields, logger.Any("pk", ctx.PrimaryKey))
	}
	if ctx.FieldCount > 0 {
		fields = append(fields, logger.Int("fields", ctx.FieldCount))
	}

	fields = append(fields, extraFields...)

	switch level {
	case LogLevelError:
		l.Error(message, fields...)
	case LogLevelWarn:
		l.Warn(message, fields...)
	case LogLevelInfo:
		l.Info(message, fields...)
	case LogLevelDebug:
		l.Debug(message, fields...)
	}
}
