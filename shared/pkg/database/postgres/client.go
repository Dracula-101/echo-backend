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
func (c *client) Create(ctx context.Context, model database.Model) (string, error) {
	fields, values := getFieldsAndValues(model)
	if len(fields) == 0 {
		return "", fmt.Errorf("no db tags found in model")
	}

	pkField := getPrimaryKeyField(model)
	filteredFields := []string{}
	filteredValues := []interface{}{}

	for i, field := range fields {
		// Skip primary key if it's empty string
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
		return "", fmt.Errorf("no fields to insert")
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
	c.logger.Debug("Create", logger.String("query", query))

	var returnedID interface{}
	if err := c.db.QueryRowContext(ctx, query, nargs...).Scan(&returnedID); err != nil {
		return "", err
	}

	if err := setPrimaryKeyValue(model, pkField, returnedID); err != nil {
		return "", err
	}

	return formatPrimaryKey(returnedID), nil
}

func (c *client) FindByID(ctx context.Context, model database.Model, id interface{}) error {
	fields := getFields(model)
	if len(fields) == 0 {
		return fmt.Errorf("no db tags found in model")
	}

	pkField := getPrimaryKeyField(model)
	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = $1",
		strings.Join(fields, ", "),
		model.TableName(),
		pkField,
	)

	c.logger.Debug("FindByID", logger.String("query", query))
	row := c.db.QueryRowContext(ctx, query, id)
	return scanStruct(row, model)
}

func (c *client) Update(ctx context.Context, model database.Model) error {
	fields, values := getFieldsAndValues(model)
	if len(fields) == 0 {
		return fmt.Errorf("no db tags found in model")
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
		return fmt.Errorf("no fields to update")
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
	c.logger.Debug("Update", logger.String("query", query))

	result, err := c.db.ExecContext(ctx, query, nargs...)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (c *client) Delete(ctx context.Context, model database.Model) error {
	pkField := getPrimaryKeyField(model)
	query := fmt.Sprintf(
		"UPDATE %s SET deleted_at = $1 WHERE %s = $2 AND deleted_at IS NULL",
		model.TableName(),
		pkField,
	)

	c.logger.Debug("Delete", logger.String("query", query))

	result, err := c.db.ExecContext(ctx, query, time.Now(), model.PrimaryKey())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (c *client) HardDelete(ctx context.Context, model database.Model) error {
	pkField := getPrimaryKeyField(model)
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s = $1",
		model.TableName(),
		pkField,
	)

	c.logger.Debug("HardDelete", logger.String("query", query))

	result, err := c.db.ExecContext(ctx, query, model.PrimaryKey())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (c *client) FindOne(ctx context.Context, model database.Model, query string, args ...interface{}) error {
	nargs := normalizeArgs(args)
	c.logger.Debug("FindOne", logger.String("query", query))
	row := c.db.QueryRowContext(ctx, query, nargs...)
	return scanStruct(row, model)
}

func (c *client) FindMany(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	nargs := normalizeArgs(args)
	c.logger.Debug("FindMany", logger.String("query", query))
	rows, err := c.db.QueryContext(ctx, query, nargs...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return scanStructs(rows, dest)
}

func (c *client) Exists(ctx context.Context, model database.Model, query string, args ...interface{}) (bool, error) {
	nargs := normalizeArgs(args)
	c.logger.Debug("Exists", logger.String("query", query))

	var exists bool
	err := c.db.QueryRowContext(ctx, query, nargs...).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (c *client) Count(ctx context.Context, model database.Model, query string, args ...interface{}) (int64, error) {
	nargs := normalizeArgs(args)
	c.logger.Debug("Count", logger.String("query", query))

	var count int64
	err := c.db.QueryRowContext(ctx, query, nargs...).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (c *client) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	nargs := normalizeArgs(args)
	c.logger.Debug("Query", logger.String("query", query))
	rows, err := c.db.QueryContext(ctx, query, nargs...)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	return &resultWrapper{result: result}, nil
}

func (c *client) Begin(ctx context.Context) (database.Transaction, error) {
	c.logger.Debug("Begin transaction")
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
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
		return nil, err
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
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return rbErr
		}
		return err
	}

	return tx.Commit()
}

func (c *client) Close() error {
	c.logger.Debug("Closing database")
	return c.db.Close()
}

func (c *client) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
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

func (t *transactionWrapper) Create(ctx context.Context, model database.Model) error {
	fields, values := getFieldsAndValues(model)
	if len(fields) == 0 {
		return fmt.Errorf("no db tags found in model")
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
	t.logger.Debug("TX Create", logger.String("query", query))

	pkField := getPrimaryKeyField(model)
	var returnedID interface{}

	err := t.tx.QueryRowContext(ctx, query, nargs...).Scan(&returnedID)
	if err != nil {
		return err
	}

	return setPrimaryKeyValue(model, pkField, returnedID)
}

func (t *transactionWrapper) FindByID(ctx context.Context, model database.Model, id interface{}) error {
	fields := getFields(model)
	if len(fields) == 0 {
		return fmt.Errorf("no db tags found in model")
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
	return scanStruct(row, model)
}

func (t *transactionWrapper) Update(ctx context.Context, model database.Model) error {
	fields, values := getFieldsAndValues(model)
	if len(fields) == 0 {
		return fmt.Errorf("no db tags found in model")
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
		return fmt.Errorf("no fields to update")
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
	t.logger.Debug("TX Update", logger.String("query", query))

	result, err := t.tx.ExecContext(ctx, query, nargs...)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (t *transactionWrapper) Delete(ctx context.Context, model database.Model) error {
	pkField := getPrimaryKeyField(model)
	query := fmt.Sprintf(
		"UPDATE %s SET deleted_at = $1 WHERE %s = $2 AND deleted_at IS NULL",
		model.TableName(),
		pkField,
	)

	t.logger.Debug("TX Delete", logger.String("query", query))

	result, err := t.tx.ExecContext(ctx, query, time.Now(), model.PrimaryKey())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (t *transactionWrapper) HardDelete(ctx context.Context, model database.Model) error {
	pkField := getPrimaryKeyField(model)
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s = $1",
		model.TableName(),
		pkField,
	)

	t.logger.Debug("TX HardDelete", logger.String("query", query))

	result, err := t.tx.ExecContext(ctx, query, model.PrimaryKey())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (t *transactionWrapper) FindOne(ctx context.Context, model database.Model, query string, args ...interface{}) error {
	nargs := normalizeArgs(args)
	t.logger.Debug("TX FindOne", logger.String("query", query))
	row := t.tx.QueryRowContext(ctx, query, nargs...)
	return scanStruct(row, model)
}

func (t *transactionWrapper) FindMany(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	nargs := normalizeArgs(args)
	t.logger.Debug("TX FindMany", logger.String("query", query))
	rows, err := t.tx.QueryContext(ctx, query, nargs...)
	if err != nil {
		return err
	}
	defer rows.Close()
	return scanStructs(rows, dest)
}

func (t *transactionWrapper) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	nargs := normalizeArgs(args)
	t.logger.Debug("TX Query", logger.String("query", query))
	rows, err := t.tx.QueryContext(ctx, query, nargs...)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	return &resultWrapper{result: result}, nil
}

func (t *transactionWrapper) Commit() error {
	return t.tx.Commit()
}

func (t *transactionWrapper) Rollback() error {
	return t.tx.Rollback()
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
func (r *rowsWrapper) ScanOne(model database.Model) error {
	if !r.rows.Next() {
		if err := r.rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	return scanStructRows(r.rows, model)
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

func (r *rowWrapper) ScanOne(model database.Model) error {
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
		return fmt.Errorf("model must be a pointer")
	}
	v = v.Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get("db") == pkField {
			fieldValue := v.Field(i)
			if !fieldValue.CanSet() {
				return fmt.Errorf("cannot set primary key field")
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
		return fmt.Errorf("dest must be a pointer to struct")
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
			} else if field.Type.Kind() == reflect.Ptr {
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
		return fmt.Errorf("dest must be a pointer to struct")
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
					return fmt.Errorf("unsupported struct type: %s", field.Type.String())
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
		return fmt.Errorf("dest must be a pointer to slice")
	}
	sliceValue := destValue.Elem()
	if sliceValue.Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to slice")
	}
	elemType := sliceValue.Type().Elem()
	isPtr := elemType.Kind() == reflect.Ptr
	if isPtr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return fmt.Errorf("slice element must be a struct or pointer to struct")
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
