package postgres

import (
	"database/sql"
	"fmt"

	"shared/pkg/database"
	"shared/pkg/logger"
)

// RowsWrapper wraps sql.Rows with logging and database interface compliance.
type RowsWrapper struct {
	rows *sql.Rows
	log  logger.Logger
}

// Next advances to the next row.
func (r *RowsWrapper) Next() bool {
	return r.rows.Next()
}

// Scan scans the current row into destination variables.
func (r *RowsWrapper) Scan(dest ...interface{}) *database.DBError {
	if err := r.rows.Scan(dest...); err != nil {
		r.log.Error("Row scan failed",
			logger.Error(err),
			logger.Int("dest_count", len(dest)),
		)
		return database.WrapDBError(err, database.CodeDBInternal, "failed to scan row")
	}
	return nil
}

// ScanModel scans the current row into a model struct.
func (r *RowsWrapper) ScanModel(model database.Model) *database.DBError {
	if err := scanStructRows(r.rows, model, r.log); err != nil {
		r.log.Error("Model scan failed",
			logger.Error(err),
			logger.String("table", model.TableName()),
		)
		return database.WrapDBError(err, database.CodeDBInternal, "failed to scan model")
	}
	return nil
}

func (r *RowsWrapper) ScanModelAndAppend(model database.Model, args ...interface{}) *database.DBError {
	if err := scanStructRowsAndAppend(r.rows, model, args...); err != nil {
		r.log.Error("Model scan and append failed",
			logger.Error(err),
			logger.String("table", model.TableName()),
		)
		return database.WrapDBError(err, database.CodeDBInternal, "failed to scan model and append")
	}
	return nil
}

// ScanAll scans all remaining rows into a slice.
func (r *RowsWrapper) ScanAll(dest interface{}) *database.DBError {
	if err := scanStructs(r.rows, dest, r.log); err != nil {
		r.log.Error("ScanAll failed",
			logger.Error(err),
			logger.String("dest_type", fmt.Sprintf("%T", dest)),
		)
		return database.WrapDBError(err, database.CodeDBInternal, "failed to scan rows")
	}
	return nil
}

// ScanOne scans exactly one row into a model, returning error if no rows exist.
func (r *RowsWrapper) ScanOne(model database.Model) *database.DBError {
	if !r.rows.Next() {
		if err := r.rows.Err(); err != nil {
			r.log.Error("Row iteration failed",
				logger.Error(err),
				logger.String("table", model.TableName()),
			)
			return database.WrapDBError(err, database.CodeDBInternal, "failed to iterate rows")
		}
		return database.NewDBError(database.CodeDBNoRows, "no rows found")
	}

	if err := scanStructRows(r.rows, model, r.log); err != nil {
		r.log.Error("ScanOne failed",
			logger.Error(err),
			logger.String("table", model.TableName()),
		)
		return database.WrapDBError(err, database.CodeDBInternal, "failed to scan row")
	}

	return nil
}

// Close closes the rows iterator.
func (r *RowsWrapper) Close() *database.DBError {
	if err := r.rows.Close(); err != nil {
		r.log.Error("Failed to close rows", logger.Error(err))
		return database.WrapDBError(err, database.CodeDBInternal, "failed to close rows")
	}
	return nil
}

// Err returns any error that occurred during iteration.
func (r *RowsWrapper) Err() *database.DBError {
	if err := r.rows.Err(); err != nil {
		r.log.Error("Rows iteration error", logger.Error(err))
		return database.WrapDBError(err, database.CodeDBInternal, "rows error")
	}
	return nil
}

// RowWrapper wraps sql.Row with logging and database interface compliance.
type RowWrapper struct {
	row *sql.Row
	log logger.Logger
}

// Scan scans the row into destination variables.
func (r *RowWrapper) Scan(dest ...any) *database.DBError {
	if err := r.row.Scan(dest...); err != nil {
		r.log.Error("Row scan failed",
			logger.Error(err),
			logger.Int("dest_count", len(dest)),
		)
		return database.WrapDBError(err, database.CodeDBInternal, "failed to scan row")
	}
	return nil
}

// ScanModel scans the row into a model struct.
func (r *RowWrapper) ScanModel(model database.Model) *database.DBError {
	return scanStruct(r.row, model)
}

// ResultWrapper wraps sql.Result with database interface compliance.
type ResultWrapper struct {
	result sql.Result
}

// LastInsertId returns the last inserted ID (not typically used with PostgreSQL).
func (r *ResultWrapper) LastInsertId() (int64, *database.DBError) {
	id, err := r.result.LastInsertId()
	if err != nil {
		return 0, database.WrapDBError(err, database.CodeDBInternal, "failed to get last insert id")
	}
	return id, nil
}

// RowsAffected returns the number of rows affected by the operation.
func (r *ResultWrapper) RowsAffected() (int64, *database.DBError) {
	count, err := r.result.RowsAffected()
	if err != nil {
		return 0, database.WrapDBError(err, database.CodeDBInternal, "failed to get rows affected")
	}
	return count, nil
}
