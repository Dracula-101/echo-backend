package postgres

import (
	"context"
	"database/sql"
	"errors"

	db "shared/pkg/database"

	"github.com/lib/pq"
)

const (
	PQCodeUniqueViolation     = "23505"
	PQCodeForeignKeyViolation = "23503"
	PQCodeNotNullViolation    = "23502"
	PQCodeCheckViolation      = "23514"
	PQCodeExclusionViolation  = "23P01"

	PQCodeSerializationFailure = "40001"
	PQCodeDeadlockDetected     = "40P01"

	PQCodeConnectionException = "08000"
	PQCodeConnectionFailure   = "08006"

	PQCodeQueryCanceled = "57014"

	PQCodeDiskFull    = "53100"
	PQCodeOutOfMemory = "53200"
)

func ConvertPQError(err error, operation, query string) error {
	if err == nil {
		return nil
	}

	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return err
	}

	if errors.Is(err, sql.ErrNoRows) {
		return db.NewDBError(db.CodeDBNoRows, "No rows found").
			WithOperation(operation).
			WithQuery(query).
			WithWrapped(err)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return db.TimeoutError(operation, err).WithQuery(query)
	}

	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return db.QueryError(query, err).WithOperation(operation)
	}

	var dbError *db.DBError

	switch pqErr.Code {
	case PQCodeUniqueViolation:
		dbError = db.NewDBError(db.CodeDBDuplicateKey, "Duplicate key violation")

	case PQCodeForeignKeyViolation:
		dbError = db.NewDBError(db.CodeDBForeignKey, "Foreign key constraint violation")

	case PQCodeNotNullViolation:
		dbError = db.NewDBError(db.CodeDBNotNull, "Not null constraint violation")

	case PQCodeCheckViolation:
		dbError = db.NewDBError(db.CodeDBCheckViolation, "Check constraint violation")

	case PQCodeExclusionViolation:
		dbError = db.NewDBError(db.CodeDBConstraint, "Exclusion constraint violation")

	case PQCodeDeadlockDetected:
		dbError = db.NewDBError(db.CodeDBDeadlock, "Deadlock detected")

	case PQCodeSerializationFailure:
		dbError = db.NewDBError(db.CodeDBSerializationFailure, "Serialization failure")

	case PQCodeConnectionException, PQCodeConnectionFailure:
		dbError = db.NewDBError(db.CodeDBConnection, "Database connection error")

	case PQCodeQueryCanceled:
		dbError = db.NewDBError(db.CodeDBTimeout, "Query canceled")

	case PQCodeDiskFull:
		dbError = db.NewDBError(db.CodeDBDiskFull, "Disk full")

	case PQCodeOutOfMemory:
		dbError = db.NewDBError(db.CodeDBOutOfMemory, "Out of memory")

	default:
		dbError = db.NewDBError(db.CodeDBInternal, pqErr.Message)
	}

	dbError.WithOperation(operation).
		WithQuery(query).
		WithSQLState(string(pqErr.Code)).
		WithWrapped(err)

	if pqErr.Table != "" {
		dbError.WithTable(pqErr.Table)
	}
	if pqErr.Column != "" {
		dbError.WithColumn(pqErr.Column)
	}
	if pqErr.Constraint != "" {
		dbError.WithConstraint(pqErr.Constraint)
	}
	if pqErr.Detail != "" {
		dbError.WithDetail("pg_detail", pqErr.Detail)
	}
	if pqErr.Hint != "" {
		dbError.WithDetail("pg_hint", pqErr.Hint)
	}
	if pqErr.Where != "" {
		dbError.WithDetail("pg_where", pqErr.Where)
	}
	if pqErr.Schema != "" {
		dbError.WithDetail("pg_schema", pqErr.Schema)
	}
	if pqErr.DataTypeName != "" {
		dbError.WithDetail("pg_datatype", pqErr.DataTypeName)
	}

	return dbError
}

func IsNoRowsError(err error) bool {
	if err == nil {
		return false
	}

	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Code() == db.CodeDBNoRows
	}
	return errors.Is(err, sql.ErrNoRows)
}

func IsDuplicateKeyError(err error) bool {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Code() == db.CodeDBDuplicateKey
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == PQCodeUniqueViolation
	}

	return false
}

func IsForeignKeyError(err error) bool {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Code() == db.CodeDBForeignKey
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == PQCodeForeignKeyViolation
	}

	return false
}

func IsNotNullError(err error) bool {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Code() == db.CodeDBNotNull
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == PQCodeNotNullViolation
	}

	return false
}

func IsCheckViolationError(err error) bool {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Code() == db.CodeDBCheckViolation
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == PQCodeCheckViolation
	}

	return false
}

func IsConstraintError(err error) bool {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		switch dbErr.Code() {
		case db.CodeDBConstraint, db.CodeDBDuplicateKey, db.CodeDBForeignKey,
			db.CodeDBNotNull, db.CodeDBCheckViolation:
			return true
		}
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case PQCodeUniqueViolation, PQCodeForeignKeyViolation,
			PQCodeNotNullViolation, PQCodeCheckViolation, PQCodeExclusionViolation:
			return true
		}
	}

	return false
}

func IsDeadlockError(err error) bool {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Code() == db.CodeDBDeadlock
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == PQCodeDeadlockDetected
	}

	return false
}

func IsSerializationError(err error) bool {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Code() == db.CodeDBSerializationFailure
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == PQCodeSerializationFailure
	}

	return false
}

func IsConnectionError(err error) bool {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Code() == db.CodeDBConnection
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == PQCodeConnectionException ||
			pqErr.Code == PQCodeConnectionFailure
	}

	return false
}

func IsTimeoutError(err error) bool {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Code() == db.CodeDBTimeout
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == PQCodeQueryCanceled
	}

	return false
}

func IsRetryableError(err error) bool {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return dbErr.IsRetryable()
	}

	return IsDeadlockError(err) || IsSerializationError(err) ||
		IsTimeoutError(err) || IsConnectionError(err)
}

func IsClientError(err error) bool {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return dbErr.IsClientError()
	}

	return IsConstraintError(err)
}

// Extraction functions for error details

func GetConstraintName(err error) string {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Constraint()
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Constraint
	}

	return ""
}

func GetTableName(err error) string {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Table()
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Table
	}

	return ""
}

func GetColumnName(err error) string {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return dbErr.Column()
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Column
	}

	return ""
}

func GetSQLState(err error) string {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		return dbErr.SQLState()
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code)
	}

	return ""
}

func ExtractPQDetails(err error) map[string]interface{} {
	var dbErr *db.DBError
	if errors.As(err, &dbErr) {
		details := make(map[string]interface{})

		for k, v := range dbErr.Details() {
			details[k] = v
		}

		if dbErr.Table() != "" {
			details["table"] = dbErr.Table()
		}
		if dbErr.Column() != "" {
			details["column"] = dbErr.Column()
		}
		if dbErr.Constraint() != "" {
			details["constraint"] = dbErr.Constraint()
		}
		if dbErr.SQLState() != "" {
			details["sql_state"] = dbErr.SQLState()
		}
		if dbErr.Operation() != "" {
			details["operation"] = dbErr.Operation()
		}

		return details
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		details := make(map[string]interface{})

		if pqErr.Table != "" {
			details["table"] = pqErr.Table
		}
		if pqErr.Column != "" {
			details["column"] = pqErr.Column
		}
		if pqErr.Constraint != "" {
			details["constraint"] = pqErr.Constraint
		}
		if pqErr.Detail != "" {
			details["detail"] = pqErr.Detail
		}
		if pqErr.Hint != "" {
			details["hint"] = pqErr.Hint
		}
		if pqErr.Where != "" {
			details["where"] = pqErr.Where
		}
		if pqErr.Schema != "" {
			details["schema"] = pqErr.Schema
		}
		if pqErr.DataTypeName != "" {
			details["datatype"] = pqErr.DataTypeName
		}
		details["sql_state"] = string(pqErr.Code)

		return details
	}

	return nil
}
