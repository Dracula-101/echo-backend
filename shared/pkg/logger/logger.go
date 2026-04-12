package logger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	pkgErrors "shared/pkg/errors"
)

type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	Request(ctx context.Context, method string, routePath string, statusCode int, duration time.Duration, bodySize int64, msg string, fields ...Field)
	With(fields ...Field) Logger
	WithContext(ctx context.Context) Logger
	Sync() error
}

type Field interface {
	Key() string
	Value() interface{}
	FormattedValue() string
}

type field struct {
	key            string
	value          interface{}
	formattedValue string
}

func (f *field) Key() string        { return f.key }
func (f *field) Value() interface{} { return f.value }

func (f *field) FormattedValue() string {
	if f.formattedValue != "" {
		return f.formattedValue
	}
	return formatValue(f.value, 0, 10)
}

const (
	maxFormatDepth = 10
	maxStringLen   = 500
	maxArrayLen    = 50
	maxMapKeys     = 50
	formatIndent   = "  "
)

func formatValue(v interface{}, depth, maxDepth int) string {
	if depth > maxDepth {
		return "<max depth exceeded>"
	}

	if v == nil {
		return "null"
	}

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return "null"
		}
		val = val.Elem()
		v = val.Interface()
	}

	switch val.Kind() {
	case reflect.Struct:
		return formatStruct(val, depth, maxDepth)
	case reflect.Map:
		return formatMap(val, depth, maxDepth)
	case reflect.Slice, reflect.Array:
		return formatSlice(val, depth, maxDepth)
	case reflect.String:
		s := val.String()
		if len(s) > maxStringLen {
			s = s[:maxStringLen-3] + "..."
		}
		return fmt.Sprintf("%q", s)
	case reflect.Bool:
		return fmt.Sprintf("%t", val.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", val.Uint())
	case reflect.Float32, reflect.Float64:
		f := val.Float()
		if f == float64(int64(f)) {
			return fmt.Sprintf("%.1f", f)
		}
		return fmt.Sprintf("%g", f)
	case reflect.Chan:
		return fmt.Sprintf("<chan %s>", val.Type().Elem().String())
	case reflect.Func:
		return "<func>"
	case reflect.Invalid:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatStruct(val reflect.Value, depth, maxDepth int) string {
	if val.Type().String() == "time.Time" {
		t := val.Interface().(time.Time)
		if t.IsZero() {
			return "null"
		}
		return fmt.Sprintf("%q", t.Format(time.RFC3339))
	}

	if val.CanInterface() {
		if err, ok := val.Interface().(error); ok {
			return fmt.Sprintf("%q", err.Error())
		}
		if stringer, ok := val.Interface().(fmt.Stringer); ok {
			return fmt.Sprintf("%q", stringer.String())
		}
	}

	typ := val.Type()
	numFields := val.NumField()
	if numFields == 0 {
		return "{}"
	}

	var visibleFields []struct {
		name  string
		value reflect.Value
	}

	for i := 0; i < numFields; i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldVal := val.Field(i)
		name := field.Name

		if tag := field.Tag.Get("json"); tag != "" {
			parts := strings.Split(tag, ",")
			if parts[0] == "-" {
				continue
			}
			if parts[0] != "" {
				name = parts[0]
			}
			hasOmitempty := false
			for _, opt := range parts[1:] {
				if opt == "omitempty" {
					hasOmitempty = true
					break
				}
			}
			if hasOmitempty && isZeroValue(fieldVal) {
				continue
			}
		}

		if isZeroValue(fieldVal) {
			continue
		}

		visibleFields = append(visibleFields, struct {
			name  string
			value reflect.Value
		}{name, fieldVal})
	}

	if len(visibleFields) == 0 {
		return "{}"
	}

	var sb strings.Builder
	baseIndent := strings.Repeat(formatIndent, depth)
	fieldIndent := strings.Repeat(formatIndent, depth+1)

	sb.WriteString("{\n")

	for i, f := range visibleFields {
		sb.WriteString(fieldIndent)
		sb.WriteString(fmt.Sprintf("%q: ", f.name))
		sb.WriteString(formatFieldValue(f.value, depth+1, maxDepth))

		if i < len(visibleFields)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(baseIndent)
	sb.WriteString("}")

	return sb.String()
}

func formatMap(val reflect.Value, depth, maxDepth int) string {
	if val.IsNil() {
		return "null"
	}

	keys := val.MapKeys()
	if len(keys) == 0 {
		return "{}"
	}

	var sb strings.Builder
	baseIndent := strings.Repeat(formatIndent, depth)
	fieldIndent := strings.Repeat(formatIndent, depth+1)

	sb.WriteString("{\n")

	displayCount := len(keys)
	truncated := false
	if displayCount > maxMapKeys {
		displayCount = maxMapKeys
		truncated = true
	}

	for i := 0; i < displayCount; i++ {
		k := keys[i]
		v := val.MapIndex(k)

		sb.WriteString(fieldIndent)
		sb.WriteString(fmt.Sprintf("%q: ", fmt.Sprintf("%v", k.Interface())))
		sb.WriteString(formatFieldValue(v, depth+1, maxDepth))

		if i < displayCount-1 || truncated {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	if truncated {
		sb.WriteString(fieldIndent)
		sb.WriteString(fmt.Sprintf("... (%d more keys)\n", len(keys)-maxMapKeys))
	}

	sb.WriteString(baseIndent)
	sb.WriteString("}")

	return sb.String()
}

func formatSlice(val reflect.Value, depth, maxDepth int) string {
	if val.Kind() == reflect.Slice && val.IsNil() {
		return "null"
	}

	length := val.Len()
	if length == 0 {
		return "[]"
	}

	if val.Type().Elem().Kind() == reflect.Uint8 {
		return fmt.Sprintf("<bytes[%d]>", length)
	}

	var sb strings.Builder
	baseIndent := strings.Repeat(formatIndent, depth)
	itemIndent := strings.Repeat(formatIndent, depth+1)

	sb.WriteString("[\n")

	displayCount := length
	truncated := false
	if displayCount > maxArrayLen {
		displayCount = maxArrayLen
		truncated = true
	}

	for i := 0; i < displayCount; i++ {
		sb.WriteString(itemIndent)
		sb.WriteString(formatFieldValue(val.Index(i), depth+1, maxDepth))

		if i < displayCount-1 || truncated {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	if truncated {
		sb.WriteString(itemIndent)
		sb.WriteString(fmt.Sprintf("... (%d more items)\n", length-maxArrayLen))
	}

	sb.WriteString(baseIndent)
	sb.WriteString("]")

	return sb.String()
}

func formatFieldValue(v reflect.Value, depth, maxDepth int) string {
	if !v.IsValid() {
		return "null"
	}

	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return "null"
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		s := v.String()
		if len(s) > maxStringLen {
			s = s[:maxStringLen-3] + "..."
		}
		return fmt.Sprintf("%q", s)
	case reflect.Struct:
		return formatStruct(v, depth, maxDepth)
	case reflect.Map:
		return formatMap(v, depth, maxDepth)
	case reflect.Slice, reflect.Array:
		return formatSlice(v, depth, maxDepth)
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		if f == float64(int64(f)) {
			return fmt.Sprintf("%.1f", f)
		}
		return fmt.Sprintf("%g", f)
	case reflect.Chan:
		return fmt.Sprintf("<chan %s>", v.Type().Elem().String())
	case reflect.Func:
		return "<func>"
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

func isZeroValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !isZeroValue(v.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Map, reflect.Slice:
		return v.IsNil() || v.Len() == 0
	case reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Struct:
		if v.Type().String() == "time.Time" {
			return v.Interface().(time.Time).IsZero()
		}
		return false
	case reflect.Chan, reflect.Func:
		return v.IsNil()
	}
	return false
}

func String(key, val string) Field {
	return &field{key: key, value: val, formattedValue: val}
}

func Strings(key string, val []string) Field {
	return &field{key: key, value: val}
}

func Int(key string, val int) Field {
	return &field{key: key, value: val, formattedValue: fmt.Sprintf("%d", val)}
}

func Int8(key string, val int8) Field {
	return &field{key: key, value: val, formattedValue: fmt.Sprintf("%d", val)}
}

func Int16(key string, val int16) Field {
	return &field{key: key, value: val, formattedValue: fmt.Sprintf("%d", val)}
}

func Int32(key string, val int32) Field {
	return &field{key: key, value: val, formattedValue: fmt.Sprintf("%d", val)}
}

func Int64(key string, val int64) Field {
	return &field{key: key, value: val, formattedValue: fmt.Sprintf("%d", val)}
}

func Uint(key string, val uint) Field {
	return &field{key: key, value: val, formattedValue: fmt.Sprintf("%d", val)}
}

func Uint8(key string, val uint8) Field {
	return &field{key: key, value: val, formattedValue: fmt.Sprintf("%d", val)}
}

func Uint16(key string, val uint16) Field {
	return &field{key: key, value: val, formattedValue: fmt.Sprintf("%d", val)}
}

func Uint32(key string, val uint32) Field {
	return &field{key: key, value: val, formattedValue: fmt.Sprintf("%d", val)}
}

func Uint64(key string, val uint64) Field {
	return &field{key: key, value: val, formattedValue: fmt.Sprintf("%d", val)}
}

func Float32(key string, val float32) Field {
	return &field{key: key, value: val, formattedValue: fmt.Sprintf("%g", val)}
}

func Float64(key string, val float64) Field {
	return &field{key: key, value: val, formattedValue: fmt.Sprintf("%g", val)}
}

func Bool(key string, val bool) Field {
	return &field{key: key, value: val, formattedValue: fmt.Sprintf("%t", val)}
}

func Any(key string, val interface{}) Field {
	return &field{key: key, value: val}
}

func Struct(key string, val interface{}) Field {
	return &field{key: key, value: val, formattedValue: formatValue(val, 0, maxFormatDepth)}
}

func JSON(key string, val interface{}) Field {
	b, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return &field{key: key, value: val, formattedValue: fmt.Sprintf("marshal error: %v", err)}
	}
	return &field{key: key, value: val, formattedValue: string(b)}
}

func Object(key string, val interface{}) Field {
	return &field{key: key, value: val, formattedValue: formatValue(val, 0, maxFormatDepth)}
}

func Array(key string, val interface{}) Field {
	return &field{key: key, value: val}
}

func Map(key string, val interface{}) Field {
	return &field{key: key, value: val}
}

func Time(key string, val time.Time) Field {
	return &field{key: key, value: val, formattedValue: val.Format(time.RFC3339)}
}

func Duration(key string, val time.Duration) Field {
	return &field{key: key, value: val, formattedValue: val.String()}
}

func Binary(key string, val []byte) Field {
	return &field{key: key, value: val, formattedValue: fmt.Sprintf("<bytes[%d]>", len(val))}
}

func Error(err error) Field {
	if isNilError(err) {
		return &field{key: "error", value: nil, formattedValue: "null"}
	}

	var appErr pkgErrors.AppError
	if errors.As(err, &appErr) {
		errMap := map[string]interface{}{
			"code":          appErr.Code(),
			"service":       appErr.Service(),
			"message":       appErr.Message(),
			"correlationId": appErr.CorrelationID(),
			"details":       appErr.Details(),
			"stackTrace":    appErr.StackTrace(),
		}
		return &field{key: "error", value: errMap}
	}
	return &field{key: "error", value: err.Error(), formattedValue: err.Error()}
}

func Err(key string, err error) Field {
	if isNilError(err) {
		return &field{key: key, value: nil, formattedValue: "null"}
	}
	return &field{key: key, value: err.Error(), formattedValue: err.Error()}
}

func isNilError(err error) bool {
	if err == nil {
		return true
	}
	v := reflect.ValueOf(err)
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface, reflect.Func, reflect.Chan:
		return v.IsNil()
	default:
		return false
	}
}

func Stack(key string, skip int) Field {
	return &field{key: key, value: "stack", formattedValue: "<stack>"}
}

type Config struct {
	Level      Level
	Output     io.Writer
	Format     Format
	TimeFormat string
	Service    string
}

type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)
