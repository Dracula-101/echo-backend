package postgres

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"shared/pkg/database"
	"shared/pkg/logger"

	"github.com/lib/pq"
)

// normalizeArgs converts slice arguments to pq.Array for PostgreSQL compatibility.
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

// getFields extracts database field names from a struct's db tags.
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
		if tag := t.Field(i).Tag.Get("db"); tag != "" && tag != "-" {
			fields = append(fields, tag)
		}
	}
	return fields
}

// getFieldsAndValues extracts database field names and their values from a struct.
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
		value := extractFieldValue(field, fieldValue)

		// Skip zero-value time fields (likely unset)
		if field.Type.String() == "time.Time" {
			if timeVal, ok := value.(time.Time); ok && timeVal.IsZero() {
				continue
			}
		}

		if value != nil || isPointerField(field) {
			fields = append(fields, tag)
			values = append(values, value)
		}
	}

	return fields, values
}

// getInsertableFieldsAndValues returns fields and values suitable for insert,
// filtering out empty primary keys.
func getInsertableFieldsAndValues(model database.Model, pkField string) ([]string, []interface{}) {
	fields, values := getFieldsAndValues(model)

	filteredFields := make([]string, 0, len(fields))
	filteredValues := make([]interface{}, 0, len(values))

	for i, field := range fields {
		// Skip empty string primary keys (auto-generated)
		if field == pkField {
			if str, ok := values[i].(string); ok && str == "" {
				continue
			}
		}
		filteredFields = append(filteredFields, field)
		filteredValues = append(filteredValues, values[i])
	}

	return filteredFields, filteredValues
}

// extractFieldValue extracts the value from a reflect.Value, handling special types.
func extractFieldValue(field reflect.StructField, fieldValue reflect.Value) interface{} {
	// Handle pointer types
	if fieldValue.Kind() == reflect.Ptr {
		if fieldValue.IsNil() {
			return nil
		}
		return fieldValue.Elem().Interface()
	}

	// Handle byte slices ([]byte)
	if fieldValue.Kind() == reflect.Slice && fieldValue.Type().Elem().Kind() == reflect.Uint8 {
		return fieldValue.Interface()
	}

	// Handle pq.StringArray
	if fieldValue.Type().String() == "pq.StringArray" {
		return fieldValue.Interface()
	}

	// Handle time.Time
	if fieldValue.Kind() == reflect.Struct && fieldValue.Type().String() == "time.Time" {
		return fieldValue.Interface().(time.Time)
	}

	return fieldValue.Interface()
}

// isPointerField checks if a struct field is a pointer type.
func isPointerField(field reflect.StructField) bool {
	return field.Type.Kind() == reflect.Ptr
}

// getPrimaryKeyField finds the primary key field name from db tags.
func getPrimaryKeyField(model interface{}) string {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get("pk") == "true" {
			return field.Tag.Get("db")
		}
	}

	return "id" // Default primary key field
}

// setPrimaryKeyValue sets the primary key field on a model after insert.
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
		if field.Tag.Get("db") != pkField {
			continue
		}

		fieldValue := v.Field(i)
		if !fieldValue.CanSet() {
			return database.NewDBError(database.CodeDBInternal, "cannot set primary key field").
				WithDetail("pk_field", pkField).
				WithDetail("field_name", field.Name)
		}

		val := reflect.ValueOf(value)
		if fieldValue.Type() != val.Type() && val.Type().ConvertibleTo(fieldValue.Type()) {
			val = val.Convert(fieldValue.Type())
		}

		fieldValue.Set(val)
		return nil
	}

	return nil
}

// formatPrimaryKey converts a primary key value to string representation.
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

// scanStruct scans a sql.Row into a struct using db tags.
func scanStruct(row *sql.Row, dest interface{}) *database.DBError {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return database.NewDBError(database.CodeDBInternal, "dest must be a pointer to struct")
	}

	dests, err := buildScanDestinations(v.Elem())
	if err != nil {
		return err
	}

	if scanErr := row.Scan(dests...); scanErr != nil {
		return database.WrapDBError(scanErr, database.CodeDBInternal, "failed to scan struct")
	}

	return nil
}

// scanStructRows scans the current row from sql.Rows into a struct.
func scanStructRows(rows *sql.Rows, dest interface{}, log logger.Logger) *database.DBError {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return database.NewDBError(database.CodeDBInternal, "dest must be a pointer to struct")
	}

	dests, err := buildScanDestinations(v.Elem())
	if err != nil {
		return err
	}

	if scanErr := rows.Scan(dests...); scanErr != nil {
		return database.WrapDBError(scanErr, database.CodeDBInternal, "failed to scan struct rows")
	}

	return nil
}

// scanStructs scans all rows into a slice of structs.
func scanStructs(rows *sql.Rows, dest interface{}, log logger.Logger) *database.DBError {
	destValue := reflect.ValueOf(dest)
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
		return database.NewDBError(database.CodeDBInternal, "slice element must be a struct")
	}

	for rows.Next() {
		elemValue := reflect.New(elemType)
		elem := elemValue.Elem()

		dests, err := buildScanDestinationsForType(elem, elemType)
		if err != nil {
			return err
		}

		if scanErr := rows.Scan(dests...); scanErr != nil {
			return database.WrapDBError(scanErr, database.CodeDBInternal, "failed to scan row")
		}

		if isPtr {
			sliceValue.Set(reflect.Append(sliceValue, elemValue))
		} else {
			sliceValue.Set(reflect.Append(sliceValue, elem))
		}
	}

	if err := rows.Err(); err != nil {
		return database.WrapDBError(err, database.CodeDBInternal, "failed to iterate rows")
	}

	return nil
}

func scanStructRowsAndAppend(rows *sql.Rows, model database.Model, args ...interface{}) *database.DBError {
	v := reflect.ValueOf(model)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return database.NewDBError(database.CodeDBInternal, "model must be a pointer to struct")
	}

	dests, err := buildScanDestinations(v.Elem())
	if err != nil {
		return err
	}

	// Append additional args to dests
	for _, arg := range args {
		dests = append(dests, arg)
	}

	if scanErr := rows.Scan(dests...); scanErr != nil {
		return database.WrapDBError(scanErr, database.CodeDBInternal, "failed to scan struct rows and append")
	}

	return nil
}

// buildScanDestinations creates scan destination pointers for a struct value.
func buildScanDestinations(v reflect.Value) ([]interface{}, *database.DBError) {
	t := v.Type()
	dests := make([]interface{}, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}

		dest, err := createScanDest(field, v.Field(i))
		if err != nil {
			return nil, err
		}
		dests = append(dests, dest)
	}

	return dests, nil
}

// buildScanDestinationsForType is similar to buildScanDestinations but uses explicit type.
func buildScanDestinationsForType(elem reflect.Value, elemType reflect.Type) ([]interface{}, *database.DBError) {
	dests := make([]interface{}, 0, elemType.NumField())

	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		if tag := field.Tag.Get("db"); tag != "" && tag != "-" {
			fieldValue := elem.Field(i)

			switch {
			case field.Type.String() == "pq.StringArray":
				dests = append(dests, pq.Array(fieldValue.Addr().Interface()))
			case field.Type.String() == "*pq.StringArray":
				dests = append(dests, fieldValue.Interface())
			default:
				dests = append(dests, fieldValue.Addr().Interface())
			}
		}
	}

	return dests, nil
}

// createScanDest creates an appropriate scan destination for a field.
func createScanDest(field reflect.StructField, fieldValue reflect.Value) (interface{}, *database.DBError) {
	typeName := field.Type.String()

	switch {
	case typeName == "pq.StringArray":
		return pq.Array(fieldValue.Addr().Interface()), nil

	case typeName == "json.RawMessage":
		// Note: original code had a bug adding this twice - fixed here
		return fieldValue.Addr().Interface(), nil

	case field.Type.Kind() == reflect.Ptr:
		return fieldValue.Addr().Interface(), nil

	case field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Uint8:
		return fieldValue.Addr().Interface(), nil

	case field.Type.Kind() == reflect.Struct:
		if typeName == "time.Time" {
			return fieldValue.Addr().Interface(), nil
		}
		return nil, database.NewDBError(database.CodeDBInternal, "unsupported struct type").
			WithDetail("type", typeName)

	default:
		return fieldValue.Addr().Interface(), nil
	}
}

// formatFieldNames formats a slice of field names for logging.
func formatFieldNames(fields []string) string {
	return strings.Join(fields, ", ")
}
