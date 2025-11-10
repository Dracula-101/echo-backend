package database

import (
	"fmt"
	"strings"
	"time"
)

type DBError struct {
	code      string
	message   string
	wrapped   error
	timestamp time.Time

	operation string
	table     string
	query     string

	column     string
	constraint string
	sqlState   string

	details map[string]interface{}
}

const (
	// Connection errors (08xxx)
	CodeDBConnection           = "DB_CONNECTION_ERROR"
	CodeDBTimeout              = "DB_TIMEOUT"
	CodeDBNoRows               = "DB_NO_ROWS"
	CodeDBDuplicateKey         = "DB_DUPLICATE_KEY"
	CodeDBForeignKey           = "DB_FOREIGN_KEY_VIOLATION"
	CodeDBConstraint           = "DB_CONSTRAINT_VIOLATION"
	CodeDBNotNull              = "DB_NOT_NULL_VIOLATION"
	CodeDBCheckViolation       = "DB_CHECK_VIOLATION"
	CodeDBDataException        = "DB_DATA_EXCEPTION"
	CodeDBTransaction          = "DB_TRANSACTION_ERROR"
	CodeDBDeadlock             = "DB_DEADLOCK"
	CodeDBSerializationFailure = "DB_SERIALIZATION_FAILURE"
	CodeDBQuery                = "DB_QUERY_ERROR"
	CodeDBInvalidInput         = "DB_INVALID_INPUT"
	CodeDBSyntaxError          = "DB_SYNTAX_ERROR"
	CodeDBPermission           = "DB_PERMISSION_DENIED"
	CodeDBInvalidAuth          = "DB_INVALID_AUTH"
	CodeDBInternal             = "DB_INTERNAL_ERROR"
	CodeDBDiskFull             = "DB_DISK_FULL"
	CodeDBOutOfMemory          = "DB_OUT_OF_MEMORY"
)

func NewDBError(code, message string) *DBError {
	return &DBError{
		code:      code,
		message:   message,
		details:   make(map[string]interface{}),
		timestamp: time.Now(),
	}
}

func (e *DBError) Error() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("[%s] %s", e.code, e.message))

	if e.operation != "" {
		parts = append(parts, fmt.Sprintf("operation=%s", e.operation))
	}

	if e.table != "" {
		parts = append(parts, fmt.Sprintf("table=%s", e.table))
	}

	if e.column != "" {
		parts = append(parts, fmt.Sprintf("column=%s", e.column))
	}

	if e.constraint != "" {
		parts = append(parts, fmt.Sprintf("constraint=%s", e.constraint))
	}

	if e.wrapped != nil {
		parts = append(parts, fmt.Sprintf("cause=%s", e.wrapped.Error()))
	}

	return strings.Join(parts, ", ")
}

func (e *DBError) Unwrap() error {
	return e.wrapped
}

func (e *DBError) Code() string                    { return e.code }
func (e *DBError) Message() string                 { return e.message }
func (e *DBError) Operation() string               { return e.operation }
func (e *DBError) Table() string                   { return e.table }
func (e *DBError) Column() string                  { return e.column }
func (e *DBError) Constraint() string              { return e.constraint }
func (e *DBError) SQLState() string                { return e.sqlState }
func (e *DBError) Query() string                   { return e.query }
func (e *DBError) Timestamp() time.Time            { return e.timestamp }
func (e *DBError) Details() map[string]interface{} { return e.details }

// Wrapped returns the underlying error
func (e *DBError) Wrapped() error {
	return e.wrapped
}

// Builder methods for chaining
func (e *DBError) WithOperation(operation string) *DBError {
	e.operation = operation
	return e
}

func (e *DBError) WithTable(table string) *DBError {
	e.table = table
	return e
}

func (e *DBError) WithColumn(column string) *DBError {
	e.column = column
	return e
}

func (e *DBError) WithConstraint(constraint string) *DBError {
	e.constraint = constraint
	return e
}

func (e *DBError) WithSQLState(sqlState string) *DBError {
	e.sqlState = sqlState
	return e
}

func (e *DBError) WithQuery(query string) *DBError {
	e.query = query
	return e
}

func (e *DBError) WithDetail(key string, value interface{}) *DBError {
	e.details[key] = value
	return e
}

func (e *DBError) WithWrapped(err error) *DBError {
	e.wrapped = err
	return e
}

func (e *DBError) IsRetryable() bool {
	switch e.code {
	case CodeDBDeadlock, CodeDBSerializationFailure, CodeDBTimeout, CodeDBConnection:
		return true
	default:
		return false
	}
}

func (e *DBError) IsClientError() bool {
	switch e.code {
	case CodeDBInvalidInput, CodeDBSyntaxError, CodeDBDuplicateKey,
		CodeDBForeignKey, CodeDBConstraint, CodeDBNotNull, CodeDBCheckViolation:
		return true
	default:
		return false
	}
}

func NotFoundError(resource, identifier string) *DBError {
	return NewDBError(CodeDBNoRows, "Resource not found").
		WithDetail("resource", resource).
		WithDetail("identifier", identifier)
}

func DuplicateError(table, column, value string) *DBError {
	return NewDBError(CodeDBDuplicateKey, "Duplicate key violation").
		WithTable(table).
		WithColumn(column).
		WithDetail("value", value)
}

func ForeignKeyError(table, column, referencedTable string) *DBError {
	return NewDBError(CodeDBForeignKey, "Foreign key constraint violation").
		WithTable(table).
		WithColumn(column).
		WithDetail("referenced_table", referencedTable)
}

func NotNullError(table, column string) *DBError {
	return NewDBError(CodeDBNotNull, "Not null constraint violation").
		WithTable(table).
		WithColumn(column)
}

func CheckViolationError(table, constraint string) *DBError {
	return NewDBError(CodeDBCheckViolation, "Check constraint violation").
		WithTable(table).
		WithConstraint(constraint)
}

func ConstraintError(table, constraint, detail string) *DBError {
	err := NewDBError(CodeDBConstraint, "Constraint violation").
		WithTable(table).
		WithConstraint(constraint)
	if detail != "" {
		err.WithDetail("detail", detail)
	}
	return err
}

func ConnectionError(message string, err error) *DBError {
	return NewDBError(CodeDBConnection, message).WithWrapped(err)
}

func TimeoutError(operation string, err error) *DBError {
	return NewDBError(CodeDBTimeout, "Operation timed out").
		WithOperation(operation).
		WithWrapped(err)
}

func DeadlockError(table, operation string) *DBError {
	return NewDBError(CodeDBDeadlock, "Deadlock detected").
		WithTable(table).
		WithOperation(operation)
}

func TransactionError(message string, err error) *DBError {
	return NewDBError(CodeDBTransaction, message).WithWrapped(err)
}

func QueryError(query string, err error) *DBError {
	return NewDBError(CodeDBQuery, "Query execution failed").
		WithQuery(query).
		WithWrapped(err)
}

func PermissionError(operation, table string) *DBError {
	return NewDBError(CodeDBPermission, "Permission denied").
		WithOperation(operation).
		WithTable(table)
}

func InternalError(message string, err error) *DBError {
	return NewDBError(CodeDBInternal, message).WithWrapped(err)
}

func WrapDBError(err error, code, message string) *DBError {
	return NewDBError(code, message).WithWrapped(err)
}
