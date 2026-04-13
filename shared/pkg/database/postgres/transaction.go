package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"shared/pkg/database"
	"shared/pkg/logger"
)

// Transaction wraps a SQL transaction with logging capabilities.
type Transaction struct {
	tx  *sql.Tx
	log *dbLogger
}

// newTransaction creates a new Transaction wrapper.
func newTransaction(tx *sql.Tx, log *dbLogger) *Transaction {
	return &Transaction{tx: tx, log: log}
}

// Insert inserts a new record within the transaction.
func (t *Transaction) Insert(ctx context.Context, model database.Model) (id *string, dbErr *database.DBError) {
	table := model.TableName()
	pkField := getPrimaryKeyField(model)

	fields, values := getFieldsAndValues(model)
	if len(fields) == 0 {
		return nil, t.log.NoFieldsError("TX:Create", table)
	}

	query := buildInsertQuery(table, fields, pkField)
	args := normalizeArgs(values)

	t.log.TxQueryStart("Create", table, len(fields))

	var returnedID string
	if err := t.tx.QueryRowContext(ctx, query, args...).Scan(&returnedID); err != nil {
		return nil, t.log.TxQueryError("Create", table, query, args, err)
	}

	if err := setPrimaryKeyValue(model, pkField, returnedID); err != nil {
		return nil, database.WrapDBError(err, database.CodeDBInternal, "failed to set primary key").
			WithDetail("table", table)
	}

	t.log.TxQuerySuccess("Create", table, formatPrimaryKey(returnedID))
	return &returnedID, nil
}

// FindByID retrieves a record by primary key within the transaction.
func (t *Transaction) FindByID(ctx context.Context, model database.Model, id interface{}) *database.DBError {
	table := model.TableName()
	fields := getFields(model)
	if len(fields) == 0 {
		return t.log.NoFieldsError("TX:FindByID", table)
	}

	query := buildSelectByIDQuery(table, fields, getPrimaryKeyField(model))

	t.log.TxQueryStart("FindByID", table, 0)

	row := t.tx.QueryRowContext(ctx, query, id)
	if err := scanStruct(row, model); err != nil {
		return t.log.TxQueryError("FindByID", table, query, []interface{}{id}, err)
	}

	t.log.TxQuerySuccess("FindByID", table, fmt.Sprint(id))
	return nil
}

// Update modifies an existing record within the transaction.
func (t *Transaction) Update(ctx context.Context, model database.Model) *database.DBError {
	table := model.TableName()
	pkField := getPrimaryKeyField(model)
	pk := model.PrimaryKey()

	setParts, updateValues := buildUpdateFields(model, pkField)
	if len(setParts) == 0 {
		return t.log.NoFieldsError("TX:Update", table)
	}

	updateValues = append(updateValues, pk)
	query := buildUpdateQuery(table, setParts, pkField, len(updateValues))
	args := normalizeArgs(updateValues)

	t.log.TxQueryStart("Update", table, len(setParts))

	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return t.log.TxQueryError("Update", table, query, args, err)
	}

	if dbErr := checkRowsAffected(result, "TX:Update", table, pk); dbErr != nil {
		return dbErr
	}

	t.log.TxQuerySuccess("Update", table, fmt.Sprint(pk))
	return nil
}

// Delete performs a soft delete within the transaction.
func (t *Transaction) Delete(ctx context.Context, model database.Model) *database.DBError {
	table := model.TableName()
	pkField := getPrimaryKeyField(model)
	pk := model.PrimaryKey()

	query := fmt.Sprintf(
		"UPDATE %s SET deleted_at = $1 WHERE %s = $2 AND deleted_at IS NULL",
		table, pkField,
	)

	t.log.TxQueryStart("Delete", table, 0)

	result, err := t.tx.ExecContext(ctx, query, time.Now(), pk)
	if err != nil {
		return t.log.TxQueryError("Delete", table, query, []interface{}{pk}, err)
	}

	if dbErr := checkRowsAffected(result, "TX:Delete", table, pk); dbErr != nil {
		return dbErr
	}

	t.log.TxQuerySuccess("Delete", table, fmt.Sprint(pk))
	return nil
}

// HardDelete permanently removes a record within the transaction.
func (t *Transaction) HardDelete(ctx context.Context, model database.Model) *database.DBError {
	table := model.TableName()
	pkField := getPrimaryKeyField(model)
	pk := model.PrimaryKey()

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1", table, pkField)

	t.log.TxQueryStart("HardDelete", table, 0)

	result, err := t.tx.ExecContext(ctx, query, pk)
	if err != nil {
		return t.log.TxQueryError("HardDelete", table, query, []interface{}{pk}, err)
	}

	if dbErr := checkRowsAffected(result, "TX:HardDelete", table, pk); dbErr != nil {
		return dbErr
	}

	t.log.TxQuerySuccess("HardDelete", table, fmt.Sprint(pk))
	return nil
}

// FindOne retrieves a single record within the transaction.
func (t *Transaction) FindOne(ctx context.Context, model database.Model, query string, args ...interface{}) *database.DBError {
	table := model.TableName()
	nargs := normalizeArgs(args)

	t.log.TxQueryStart("FindOne", table, 0)

	row := t.tx.QueryRowContext(ctx, query, nargs...)
	if err := scanStruct(row, model); err != nil {
		return t.log.TxQueryError("FindOne", table, query, nargs, err)
	}

	t.log.TxQuerySuccess("FindOne", table, "")
	return nil
}

// FindMany retrieves multiple records within the transaction.
func (t *Transaction) FindMany(ctx context.Context, dest interface{}, query string, args ...interface{}) *database.DBError {
	nargs := normalizeArgs(args)

	t.log.Debug("TX:FindMany executing", logger.String("query", query))

	rows, err := t.tx.QueryContext(ctx, query, nargs...)
	if err != nil {
		return t.log.TxQueryError("FindMany", "", query, nargs, err)
	}
	defer rows.Close()

	if err := scanStructs(rows, dest, t.log.Logger); err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to scan results").
			WithDetail("operation", "TX:FindMany")
	}

	t.log.Debug("TX:FindMany completed")
	return nil
}

// Query executes a raw query within the transaction.
func (t *Transaction) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, *database.DBError) {
	nargs := normalizeArgs(args)

	t.log.Debug("TX:Query executing", logger.String("query", query))

	rows, err := t.tx.QueryContext(ctx, query, nargs...)
	if err != nil {
		return nil, t.log.TxQueryError("Query", "", query, nargs, err)
	}

	return &RowsWrapper{rows: rows, log: t.log.Logger}, nil
}

// QueryRow executes a query expected to return a single row within the transaction.
func (t *Transaction) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	nargs := normalizeArgs(args)

	t.log.Debug("TX:QueryRow executing", logger.String("query", query))

	return &RowWrapper{row: t.tx.QueryRowContext(ctx, query, nargs...), log: t.log.Logger}
}

// Exec executes a query that doesn't return rows within the transaction.
func (t *Transaction) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, *database.DBError) {
	nargs := normalizeArgs(args)

	t.log.Debug("TX:Exec executing", logger.String("query", query))

	result, err := t.tx.ExecContext(ctx, query, nargs...)
	if err != nil {
		return nil, t.log.TxQueryError("Exec", "", query, nargs, err)
	}

	return &ResultWrapper{result: result}, nil
}

// Commit commits the transaction.
func (t *Transaction) Commit() *database.DBError {
	if err := t.tx.Commit(); err != nil {
		t.log.Error("Transaction commit failed", logger.Error(err))
		return database.WrapDBError(err, database.CodeDBInternal, "failed to commit transaction")
	}

	t.log.Debug("Transaction committed")
	return nil
}

// Rollback rolls back the transaction.
func (t *Transaction) Rollback() *database.DBError {
	if err := t.tx.Rollback(); err != nil {
		t.log.Error("Transaction rollback failed", logger.Error(err))
		return database.WrapDBError(err, database.CodeDBInternal, "failed to rollback transaction")
	}

	t.log.Debug("Transaction rolled back")
	return nil
}

// buildTxLogFields creates common log fields for transaction operations.
func buildTxLogFields(operation, table string, additional ...logger.Field) []logger.Field {
	fields := []logger.Field{
		logger.String("operation", "TX:"+operation),
		logger.String("table", table),
	}
	return append(fields, additional...)
}

// formatArgTypes returns a comma-separated list of argument types for debugging.
func formatArgTypes(args []interface{}) string {
	if len(args) == 0 {
		return ""
	}

	types := make([]string, len(args))
	for i, arg := range args {
		types[i] = fmt.Sprintf("%T", arg)
	}
	return strings.Join(types, ", ")
}
