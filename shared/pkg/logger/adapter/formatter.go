package adapter

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"
)

// FormatterConfig holds configuration for value formatting
type FormatterConfig struct {
	MaxDepth          int
	MaxArrayDisplay   int
	MaxStringDisplay  int
	MaxMapKeys        int
	IndentSize        int
	ColorizeOutput    bool
	SanitizeSensitive bool
}

// DefaultFormatterConfig returns the default formatter configuration
func DefaultFormatterConfig() FormatterConfig {
	return FormatterConfig{
		MaxDepth:          15,
		MaxArrayDisplay:   100,
		MaxStringDisplay:  1000,
		MaxMapKeys:        100,
		IndentSize:        2,
		ColorizeOutput:    true,
		SanitizeSensitive: true,
	}
}

// AdvancedFormatter provides advanced formatting capabilities with deep struct support
type AdvancedFormatter struct {
	config       FormatterConfig
	sensitiveMap map[string]bool
	visitedAddrs map[uintptr]bool
	colorScheme  ColorScheme
}

// ColorScheme defines colors for different types
type ColorScheme struct {
	KeyColor        string
	StringColor     string
	NumberColor     string
	BoolColor       string
	NullColor       string
	ErrorColor      string
	TypeColor       string
	BracketColor    string
	CommaColor      string
	StructNameColor string
	FieldNameColor  string
	QuoteColor      string
}

// DefaultColorScheme returns a light, pleasant color scheme
func DefaultColorScheme() ColorScheme {
	return ColorScheme{
		KeyColor:        "\x1b[38;5;33m",  // Light blue for keys
		StringColor:     "\x1b[38;5;78m",  // Light green for strings
		NumberColor:     "\x1b[38;5;214m", // Light orange for numbers
		BoolColor:       "\x1b[38;5;213m", // Light pink/magenta for booleans
		NullColor:       "\x1b[38;5;245m", // Light gray for null
		ErrorColor:      "\x1b[38;5;204m", // Light red for errors
		TypeColor:       "\x1b[38;5;141m", // Light purple for types
		BracketColor:    "\x1b[38;5;252m", // Very light gray for brackets
		CommaColor:      "\x1b[38;5;250m", // Light gray for commas
		StructNameColor: "\x1b[38;5;117m", // Light cyan for struct names
		FieldNameColor:  "\x1b[38;5;111m", // Light blue-cyan for field names
		QuoteColor:      "\x1b[38;5;249m", // Very light gray for quotes
	}
}

// NewAdvancedFormatter creates a new advanced formatter with the given config
func NewAdvancedFormatter(config FormatterConfig) *AdvancedFormatter {
	return &AdvancedFormatter{
		config:       config,
		sensitiveMap: sensitiveFields,
		visitedAddrs: make(map[uintptr]bool),
		colorScheme:  DefaultColorScheme(),
	}
}

// Format formats a value with advanced struct support
func (af *AdvancedFormatter) Format(v interface{}) string {
	af.visitedAddrs = make(map[uintptr]bool)
	return af.formatValue(reflect.ValueOf(v), 0, "")
}

// formatValue is the main formatting dispatcher
func (af *AdvancedFormatter) formatValue(val reflect.Value, depth int, fieldName string) string {
	if depth > af.config.MaxDepth {
		return af.colorize("<max depth reached>", af.colorScheme.TypeColor)
	}

	if !val.IsValid() {
		return af.colorize("null", af.colorScheme.NullColor)
	}

	// Handle pointers and interfaces
	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return af.colorize("null", af.colorScheme.NullColor)
		}

		// Check for circular references
		if val.Kind() == reflect.Ptr {
			addr := val.Pointer()
			if af.visitedAddrs[addr] {
				return af.colorize("<circular reference>", af.colorScheme.TypeColor)
			}
			af.visitedAddrs[addr] = true
			defer func() { delete(af.visitedAddrs, addr) }()
		}

		val = val.Elem()
	}

	// Check if field is sensitive
	if fieldName != "" && af.config.SanitizeSensitive && af.isSensitiveField(fieldName) {
		return af.colorize(fmt.Sprintf("%q", sanitizationMask), af.colorScheme.ErrorColor)
	}

	switch val.Kind() {
	case reflect.Struct:
		return af.formatStruct(val, depth)
	case reflect.Map:
		return af.formatMap(val, depth)
	case reflect.Slice, reflect.Array:
		return af.formatSlice(val, depth)
	case reflect.String:
		return af.formatString(val.String())
	case reflect.Bool:
		return af.formatBool(val.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return af.formatInt(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return af.formatUint(val.Uint())
	case reflect.Float32, reflect.Float64:
		return af.formatFloat(val.Float())
	case reflect.Chan:
		return af.colorize(fmt.Sprintf("<chan %s>", val.Type().Elem().String()), af.colorScheme.TypeColor)
	case reflect.Func:
		return af.colorize("<func>", af.colorScheme.TypeColor)
	case reflect.UnsafePointer:
		return af.colorize("<unsafe.Pointer>", af.colorScheme.TypeColor)
	default:
		return fmt.Sprintf("%v", val.Interface())
	}
}

// formatStruct formats a struct with nested struct support
func (af *AdvancedFormatter) formatStruct(val reflect.Value, depth int) string {
	typ := val.Type()
	typeName := typ.String()

	// Special handling for time.Time
	if typeName == "time.Time" {
		t := val.Interface().(time.Time)
		if t.IsZero() {
			return af.colorize("null", af.colorScheme.NullColor)
		}
		return af.formatString(t.Format(time.RFC3339Nano))
	}

	// Special handling for errors
	if val.CanInterface() {
		iface := val.Interface()
		if err, ok := iface.(error); ok {
			return af.colorize(fmt.Sprintf("%q", err.Error()), af.colorScheme.ErrorColor)
		}
		if stringer, ok := iface.(fmt.Stringer); ok {
			return af.formatString(stringer.String())
		}
		if jsonMarshaler, ok := iface.(json.Marshaler); ok {
			data, err := jsonMarshaler.MarshalJSON()
			if err == nil {
				return string(data)
			}
		}
	}

	numFields := val.NumField()
	if numFields == 0 {
		return af.colorize("{}", af.colorScheme.BracketColor)
	}

	var sb strings.Builder
	indent := strings.Repeat(" ", depth*af.config.IndentSize)
	nextIndent := strings.Repeat(" ", (depth+1)*af.config.IndentSize)

	// Add struct type name
	structName := af.getShortTypeName(typeName)
	sb.WriteString(af.colorize(structName, af.colorScheme.StructNameColor))
	sb.WriteString(" ")
	sb.WriteString(af.colorize("{", af.colorScheme.BracketColor))
	sb.WriteString("\n")

	fieldCount := 0
	for i := 0; i < numFields; i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldVal := val.Field(i)
		fieldName := field.Name

		// Handle JSON tags
		jsonTag, omitEmpty := af.parseJSONTag(field.Tag.Get("json"))
		if jsonTag == "-" {
			continue
		}
		if jsonTag != "" {
			fieldName = jsonTag
		}
		if omitEmpty && af.isZeroValue(fieldVal) {
			continue
		}

		// Skip zero values for cleaner output
		if af.isZeroValue(fieldVal) && !af.isImportantField(field) {
			continue
		}

		if fieldCount > 0 {
			sb.WriteString(af.colorize(",", af.colorScheme.CommaColor))
			sb.WriteString("\n")
		}

		sb.WriteString(nextIndent)
		sb.WriteString(af.colorize(fmt.Sprintf("%q", fieldName), af.colorScheme.FieldNameColor))
		sb.WriteString(": ")
		sb.WriteString(af.formatValue(fieldVal, depth+1, fieldName))

		fieldCount++
	}

	if fieldCount > 0 {
		sb.WriteString("\n")
		sb.WriteString(indent)
	}
	sb.WriteString(af.colorize("}", af.colorScheme.BracketColor))

	return sb.String()
}

// formatMap formats a map with proper indentation
func (af *AdvancedFormatter) formatMap(val reflect.Value, depth int) string {
	if val.IsNil() {
		return af.colorize("null", af.colorScheme.NullColor)
	}

	keys := val.MapKeys()
	if len(keys) == 0 {
		return af.colorize("{}", af.colorScheme.BracketColor)
	}

	var sb strings.Builder
	indent := strings.Repeat(" ", depth*af.config.IndentSize)
	nextIndent := strings.Repeat(" ", (depth+1)*af.config.IndentSize)

	sb.WriteString(af.colorize("{", af.colorScheme.BracketColor))
	sb.WriteString("\n")

	displayCount := len(keys)
	truncated := false
	if displayCount > af.config.MaxMapKeys {
		displayCount = af.config.MaxMapKeys
		truncated = true
	}

	for i := 0; i < displayCount; i++ {
		if i > 0 {
			sb.WriteString(af.colorize(",", af.colorScheme.CommaColor))
			sb.WriteString("\n")
		}

		k := keys[i]
		v := val.MapIndex(k)

		sb.WriteString(nextIndent)
		keyStr := fmt.Sprintf("%v", k.Interface())
		sb.WriteString(af.colorize(fmt.Sprintf("%q", keyStr), af.colorScheme.KeyColor))
		sb.WriteString(": ")
		sb.WriteString(af.formatValue(v, depth+1, keyStr))
	}

	if truncated {
		sb.WriteString(af.colorize(",", af.colorScheme.CommaColor))
		sb.WriteString("\n")
		sb.WriteString(nextIndent)
		sb.WriteString(af.colorize(fmt.Sprintf("... (%d more keys)", len(keys)-displayCount), af.colorScheme.TypeColor))
	}

	sb.WriteString("\n")
	sb.WriteString(indent)
	sb.WriteString(af.colorize("}", af.colorScheme.BracketColor))

	return sb.String()
}

// formatSlice formats a slice or array
func (af *AdvancedFormatter) formatSlice(val reflect.Value, depth int) string {
	if val.Kind() == reflect.Slice && val.IsNil() {
		return af.colorize("null", af.colorScheme.NullColor)
	}

	length := val.Len()
	if length == 0 {
		return af.colorize("[]", af.colorScheme.BracketColor)
	}

	// Special handling for byte slices
	if val.Type().Elem().Kind() == reflect.Uint8 {
		return af.colorize(fmt.Sprintf("<bytes[%d]>", length), af.colorScheme.TypeColor)
	}

	var sb strings.Builder
	indent := strings.Repeat(" ", depth*af.config.IndentSize)
	nextIndent := strings.Repeat(" ", (depth+1)*af.config.IndentSize)

	sb.WriteString(af.colorize("[", af.colorScheme.BracketColor))
	sb.WriteString("\n")

	displayCount := length
	truncated := false
	if displayCount > af.config.MaxArrayDisplay {
		displayCount = af.config.MaxArrayDisplay
		truncated = true
	}

	for i := 0; i < displayCount; i++ {
		if i > 0 {
			sb.WriteString(af.colorize(",", af.colorScheme.CommaColor))
			sb.WriteString("\n")
		}

		sb.WriteString(nextIndent)
		sb.WriteString(af.formatValue(val.Index(i), depth+1, ""))
	}

	if truncated {
		sb.WriteString(af.colorize(",", af.colorScheme.CommaColor))
		sb.WriteString("\n")
		sb.WriteString(nextIndent)
		sb.WriteString(af.colorize(fmt.Sprintf("... (%d more items)", length-displayCount), af.colorScheme.TypeColor))
	}

	sb.WriteString("\n")
	sb.WriteString(indent)
	sb.WriteString(af.colorize("]", af.colorScheme.BracketColor))

	return sb.String()
}

// formatString formats a string value
func (af *AdvancedFormatter) formatString(s string) string {
	if len(s) > af.config.MaxStringDisplay {
		s = s[:af.config.MaxStringDisplay-3] + "..."
	}

	// Escape control characters
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	s = strings.ReplaceAll(s, "\"", `\"`)

	return af.colorize(af.colorScheme.QuoteColor+"\"", af.colorScheme.QuoteColor) +
		af.colorize(s, af.colorScheme.StringColor) +
		af.colorize("\"", af.colorScheme.QuoteColor)
}

// formatBool formats a boolean value
func (af *AdvancedFormatter) formatBool(b bool) string {
	return af.colorize(fmt.Sprintf("%t", b), af.colorScheme.BoolColor)
}

// formatInt formats an integer value
func (af *AdvancedFormatter) formatInt(i int64) string {
	return af.colorize(fmt.Sprintf("%d", i), af.colorScheme.NumberColor)
}

// formatUint formats an unsigned integer value
func (af *AdvancedFormatter) formatUint(u uint64) string {
	return af.colorize(fmt.Sprintf("%d", u), af.colorScheme.NumberColor)
}

// formatFloat formats a floating-point value
func (af *AdvancedFormatter) formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return af.colorize(fmt.Sprintf("%.1f", f), af.colorScheme.NumberColor)
	}
	return af.colorize(fmt.Sprintf("%g", f), af.colorScheme.NumberColor)
}

// colorize applies color if colorization is enabled
func (af *AdvancedFormatter) colorize(s, color string) string {
	if !af.config.ColorizeOutput {
		return s
	}
	return color + s + ansiReset
}

// isSensitiveField checks if a field name is sensitive
func (af *AdvancedFormatter) isSensitiveField(name string) bool {
	lowerName := strings.ToLower(name)
	return af.sensitiveMap[lowerName]
}

// isZeroValue checks if a value is a zero value
func (af *AdvancedFormatter) isZeroValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !af.isZeroValue(v.Index(i)) {
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
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Struct:
		if v.Type().String() == "time.Time" {
			return v.Interface().(time.Time).IsZero()
		}
		// For structs, check all fields
		for i := 0; i < v.NumField(); i++ {
			if !af.isZeroValue(v.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Chan, reflect.Func:
		return v.IsNil()
	}
	return false
}

// isImportantField determines if a field should always be shown even if zero
func (af *AdvancedFormatter) isImportantField(field reflect.StructField) bool {
	// Check for "important" tag
	if tag := field.Tag.Get("log"); tag != "" {
		return strings.Contains(tag, "important")
	}
	return false
}

// parseJSONTag parses a JSON tag and returns the name and omitempty flag
func (af *AdvancedFormatter) parseJSONTag(tag string) (name string, omitEmpty bool) {
	if tag == "" {
		return "", false
	}

	parts := strings.Split(tag, ",")
	name = parts[0]

	for i := 1; i < len(parts); i++ {
		if parts[i] == "omitempty" {
			omitEmpty = true
			break
		}
	}

	return name, omitEmpty
}

// getShortTypeName returns a shortened type name for cleaner output
func (af *AdvancedFormatter) getShortTypeName(typeName string) string {
	// Remove package path
	parts := strings.Split(typeName, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return typeName
}

// CompactFormatter provides compact single-line formatting
type CompactFormatter struct {
	maxLength   int
	colorize    bool
	colorScheme ColorScheme
}

// NewCompactFormatter creates a new compact formatter
func NewCompactFormatter(maxLength int, colorize bool) *CompactFormatter {
	return &CompactFormatter{
		maxLength:   maxLength,
		colorize:    colorize,
		colorScheme: DefaultColorScheme(),
	}
}

// Format formats a value in a compact single-line format
func (cf *CompactFormatter) Format(v interface{}) string {
	val := reflect.ValueOf(v)
	result := cf.formatCompact(val)

	if len(result) > cf.maxLength {
		result = result[:cf.maxLength-3] + "..."
	}

	return result
}

// formatCompact formats a value compactly
func (cf *CompactFormatter) formatCompact(val reflect.Value) string {
	if !val.IsValid() {
		return cf.color("null", cf.colorScheme.NullColor)
	}

	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return cf.color("null", cf.colorScheme.NullColor)
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		return cf.formatStructCompact(val)
	case reflect.Map:
		return cf.formatMapCompact(val)
	case reflect.Slice, reflect.Array:
		return cf.formatSliceCompact(val)
	case reflect.String:
		s := val.String()
		if len(s) > 50 {
			s = s[:47] + "..."
		}
		return cf.color(fmt.Sprintf("%q", s), cf.colorScheme.StringColor)
	case reflect.Bool:
		return cf.color(fmt.Sprintf("%t", val.Bool()), cf.colorScheme.BoolColor)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cf.color(fmt.Sprintf("%d", val.Int()), cf.colorScheme.NumberColor)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return cf.color(fmt.Sprintf("%d", val.Uint()), cf.colorScheme.NumberColor)
	case reflect.Float32, reflect.Float64:
		return cf.color(fmt.Sprintf("%g", val.Float()), cf.colorScheme.NumberColor)
	default:
		return fmt.Sprintf("%v", val.Interface())
	}
}

// formatStructCompact formats a struct compactly
func (cf *CompactFormatter) formatStructCompact(val reflect.Value) string {
	typ := val.Type()

	if typ.String() == "time.Time" {
		t := val.Interface().(time.Time)
		if t.IsZero() {
			return cf.color("null", cf.colorScheme.NullColor)
		}
		return cf.color(fmt.Sprintf("%q", t.Format(time.RFC3339)), cf.colorScheme.StringColor)
	}

	var parts []string
	for i := 0; i < val.NumField() && i < 3; i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldVal := val.Field(i)
		if isZeroValue(fieldVal) {
			continue
		}

		parts = append(parts, fmt.Sprintf("%s:%s", field.Name, cf.formatCompact(fieldVal)))
	}

	result := "{" + strings.Join(parts, " ") + "}"
	if val.NumField() > 3 {
		result += "..."
	}

	return result
}

// formatMapCompact formats a map compactly
func (cf *CompactFormatter) formatMapCompact(val reflect.Value) string {
	if val.IsNil() {
		return cf.color("null", cf.colorScheme.NullColor)
	}

	keys := val.MapKeys()
	if len(keys) == 0 {
		return "{}"
	}

	var parts []string
	for i := 0; i < len(keys) && i < 3; i++ {
		k := keys[i]
		v := val.MapIndex(k)
		parts = append(parts, fmt.Sprintf("%v:%s", k.Interface(), cf.formatCompact(v)))
	}

	result := "{" + strings.Join(parts, " ") + "}"
	if len(keys) > 3 {
		result += "..."
	}

	return result
}

// formatSliceCompact formats a slice compactly
func (cf *CompactFormatter) formatSliceCompact(val reflect.Value) string {
	if val.Kind() == reflect.Slice && val.IsNil() {
		return cf.color("null", cf.colorScheme.NullColor)
	}

	length := val.Len()
	if length == 0 {
		return "[]"
	}

	if val.Type().Elem().Kind() == reflect.Uint8 {
		return fmt.Sprintf("<bytes[%d]>", length)
	}

	var parts []string
	for i := 0; i < length && i < 3; i++ {
		parts = append(parts, cf.formatCompact(val.Index(i)))
	}

	result := "[" + strings.Join(parts, " ") + "]"
	if length > 3 {
		result += "..."
	}

	return result
}

// color applies color if enabled
func (cf *CompactFormatter) color(s, color string) string {
	if !cf.colorize {
		return s
	}
	return color + s + ansiReset
}

// StripANSI removes ANSI color codes from a string
func StripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

// MeasureVisibleLength returns the visible length of a string (without ANSI codes)
func MeasureVisibleLength(s string) int {
	return utf8.RuneCountInString(StripANSI(s))
}

// TruncateVisible truncates a string to a visible length (accounting for ANSI codes)
func TruncateVisible(s string, maxLen int) string {
	stripped := StripANSI(s)
	if utf8.RuneCountInString(stripped) <= maxLen {
		return s
	}

	// Find ANSI codes in original string
	ansiMatches := ansiPattern.FindAllStringIndex(s, -1)

	visibleCount := 0
	bytePos := 0

	for bytePos < len(s) {
		// Check if we're at an ANSI code
		inAnsi := false
		for _, match := range ansiMatches {
			if bytePos >= match[0] && bytePos < match[1] {
				inAnsi = true
				bytePos = match[1]
				break
			}
		}

		if inAnsi {
			continue
		}

		_, size := utf8.DecodeRuneInString(s[bytePos:])
		bytePos += size
		visibleCount++

		if visibleCount >= maxLen-3 {
			return s[:bytePos] + "..."
		}
	}

	return s
}
