package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"shared/pkg/database"

	"github.com/lib/pq"
)

// PostgreSQL error code classes
const (
	// Class 23 - Integrity Constraint Violation
	pqClassIntegrityConstraint = "23"

	// Specific error codes
	pqCodeUniqueViolation     = "23505"
	pqCodeForeignKeyViolation = "23503"
	pqCodeNotNullViolation    = "23502"
	pqCodeCheckViolation      = "23514"
)

// wrapDatabaseError converts a raw error into a structured DBError with context.
func wrapDatabaseError(err error, operation, table, query string) *database.DBError {
	if err == nil {
		return nil
	}

	// Already a DBError, return as-is
	var dbErr *database.DBError
	if errors.As(err, &dbErr) {
		return dbErr
	}

	// Standard SQL errors
	if sqlErr := handleSQLError(err, operation, table, query); sqlErr != nil {
		return sqlErr
	}

	// Context errors (timeout, cancellation)
	if ctxErr := handleContextError(err, operation, table, query); ctxErr != nil {
		return ctxErr
	}

	// Connection errors
	if connErr := handleConnectionError(err, operation, table); connErr != nil {
		return connErr
	}

	// PostgreSQL-specific errors
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return convertPQError(pqErr, operation, table, query)
	}

	// Generic fallback
	return database.NewDBError(database.CodeDBInternal, "database operation failed").
		WithOperation(operation).
		WithTable(table).
		WithQuery(query).
		WithWrapped(err)
}

// handleSQLError handles standard database/sql errors.
func handleSQLError(err error, operation, table, query string) *database.DBError {
	switch err {
	case sql.ErrNoRows:
		return database.NewDBError(database.CodeDBNoRows, "no rows found").
			WithOperation(operation).
			WithTable(table).
			WithQuery(query).
			WithWrapped(err)

	case sql.ErrTxDone:
		return database.NewDBError(database.CodeDBTransaction, "transaction already completed").
			WithOperation(operation).
			WithTable(table).
			WithWrapped(err)

	case sql.ErrConnDone:
		return database.NewDBError(database.CodeDBConnection, "connection already closed").
			WithOperation(operation).
			WithTable(table).
			WithWrapped(err)
	}

	return nil
}

// handleContextError handles context-related errors.
func handleContextError(err error, operation, table, query string) *database.DBError {
	if errors.Is(err, context.DeadlineExceeded) {
		return database.NewDBError(database.CodeDBTimeout, "operation timed out").
			WithOperation(operation).
			WithTable(table).
			WithQuery(query).
			WithWrapped(err)
	}

	if errors.Is(err, context.Canceled) {
		return database.NewDBError(database.CodeDBTimeout, "operation canceled").
			WithOperation(operation).
			WithTable(table).
			WithQuery(query).
			WithWrapped(err)
	}

	return nil
}

// handleConnectionError handles network/connection errors.
func handleConnectionError(err error, operation, table string) *database.DBError {
	errMsg := err.Error()

	connectionIndicators := []string{
		"connection refused",
		"no such host",
		"network is unreachable",
		"connection reset",
		"broken pipe",
		"dial tcp",
	}

	for _, indicator := range connectionIndicators {
		if strings.Contains(errMsg, indicator) {
			return database.NewDBError(database.CodeDBConnection, "connection failed").
				WithOperation(operation).
				WithTable(table).
				WithWrapped(err)
		}
	}

	return nil
}

// convertPQError converts a PostgreSQL error to a structured DBError.
func convertPQError(pqErr *pq.Error, operation, table, query string) *database.DBError {
	code := string(pqErr.Code)

	// Handle by specific error code first
	switch code {
	case pqCodeUniqueViolation:
		return database.NewDBError(database.CodeDBDuplicate, "duplicate key violation").
			WithOperation(operation).
			WithTable(table).
			WithQuery(query).
			WithDetail("constraint", pqErr.Constraint).
			WithDetail("column", pqErr.Column).
			WithWrapped(pqErr)

	case pqCodeForeignKeyViolation:
		return database.NewDBError(database.CodeDBConstraint, "foreign key violation").
			WithOperation(operation).
			WithTable(table).
			WithQuery(query).
			WithDetail("constraint", pqErr.Constraint).
			WithDetail("detail", pqErr.Detail).
			WithWrapped(pqErr)

	case pqCodeNotNullViolation:
		return database.NewDBError(database.CodeDBConstraint, "not null violation").
			WithOperation(operation).
			WithTable(table).
			WithQuery(query).
			WithDetail("column", pqErr.Column).
			WithWrapped(pqErr)

	case pqCodeCheckViolation:
		return database.NewDBError(database.CodeDBConstraint, "check constraint violation").
			WithOperation(operation).
			WithTable(table).
			WithQuery(query).
			WithDetail("constraint", pqErr.Constraint).
			WithWrapped(pqErr)
	}

	// Handle by error class
	switch pqErr.Code.Class() {
	case pqClassIntegrityConstraint:
		return database.NewDBError(database.CodeDBConstraint, "integrity constraint violation").
			WithOperation(operation).
			WithTable(table).
			WithQuery(query).
			WithDetail("constraint", pqErr.Constraint).
			WithWrapped(pqErr)

	case "08": // Connection Exception
		return database.NewDBError(database.CodeDBConnection, "connection exception").
			WithOperation(operation).
			WithTable(table).
			WithWrapped(pqErr)

	case "42": // Syntax Error or Access Rule Violation
		return database.NewDBError(database.CodeDBInternal, "syntax or access error").
			WithOperation(operation).
			WithTable(table).
			WithQuery(query).
			WithDetail("pg_message", pqErr.Message).
			WithWrapped(pqErr)

	case "53": // Insufficient Resources
		return database.NewDBError(database.CodeDBInternal, "insufficient resources").
			WithOperation(operation).
			WithTable(table).
			WithDetail("pg_message", pqErr.Message).
			WithWrapped(pqErr)

	case "57": // Operator Intervention
		return database.NewDBError(database.CodeDBConnection, "operator intervention").
			WithOperation(operation).
			WithTable(table).
			WithDetail("pg_message", pqErr.Message).
			WithWrapped(pqErr)

	case "58": // System Error
		return database.NewDBError(database.CodeDBInternal, "system error").
			WithOperation(operation).
			WithTable(table).
			WithDetail("pg_message", pqErr.Message).
			WithWrapped(pqErr)
	}

	// Default PostgreSQL error
	return database.NewDBError(database.CodeDBInternal, pqErr.Message).
		WithOperation(operation).
		WithTable(table).
		WithQuery(query).
		WithDetail("pg_code", code).
		WithDetail("pg_severity", pqErr.Severity).
		WithWrapped(pqErr)
}

// ConvertPQError is exported for use by other packages that need to convert pq errors.
func ConvertPQError(err error, operation, query string) error {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return err
	}

	return convertPQError(pqErr, operation, "", query)
}

// IsUniqueViolation checks if an error is a unique constraint violation.
func IsUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == pqCodeUniqueViolation
	}

	var dbErr *database.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Code() == database.CodeDBDuplicate
	}

	return false
}

// IsForeignKeyViolation checks if an error is a foreign key violation.
func IsForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code) == pqCodeForeignKeyViolation
	}

	return false
}

// IsNotFoundError checks if an error indicates no rows were found.
func IsNotFoundError(err error) bool {
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}

	var dbErr *database.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Code() == database.CodeDBNoRows
	}

	return false
}

// IsConnectionError checks if an error is connection-related.
func IsConnectionError(err error) bool {
	var dbErr *database.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Code() == database.CodeDBConnection
	}

	return false
}

// IsTimeoutError checks if an error is timeout-related.
func IsTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	var dbErr *database.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Code() == database.CodeDBTimeout
	}

	return false
}
