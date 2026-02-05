package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"

	"shared/pkg/logger"
)

type ColumnConfig struct {
	Start int
	Width int
}

type BoxDimensions struct {
	Width     int
	Timestamp ColumnConfig
	Level     ColumnConfig
	File      ColumnConfig
	Service   ColumnConfig
	Method    ColumnConfig
	Status    ColumnConfig
	Duration  ColumnConfig
	BodySize  ColumnConfig
	RoutePath ColumnConfig
	Message   ColumnConfig
}

const (
	timestampColumnWidth = 23
	levelColumnWidth     = 6
	fileColumnWidth      = 15
	serviceColumnWidth   = 15
	methodColumnWidth    = 4
	statusColumnWidth    = 3
	durationColumnWidth  = 10
	bodySizeColumnWidth  = 8
	routePathColumnWidth = 25

	minTerminalWidth     = 80
	defaultTerminalWidth = 220
	columnSeparatorWidth = 3
	boxBorderWidth       = 2

	callerSkipDefault   = 1
	callerSkipFormatLog = 2
	callerSkipError     = 3

	truncationSuffix   = "..."
	truncationMinWidth = 3

	ansiReset      = "\x1b[0m"
	ansiGreen      = "\x1b[32m"
	ansiCyan       = "\x1b[36m"
	ansiYellow     = "\x1b[33m"
	ansiRed        = "\x1b[31m"
	ansiWhite      = "\x1b[37m"
	ansiBlue       = "\x1b[34m"
	ansiBrightCyan = "\x1b[96m"
	ansiBold       = "\x1b[1m"
	ansiDim        = "\x1b[2m"
	ansiMagenta    = "\x1b[35m"
	ansiGray       = "\x1b[90m"

	bytesPerKilobyte     = 1024
	sensitiveFieldPrefix = 4
	sanitizationMask     = "****"

	maxDepth         = 10
	maxArrayDisplay  = 50
	maxStringDisplay = 500
	maxMapKeys       = 50
	jsonIndent       = "  "
)

var (
	ansiPattern    = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	whitespaceNorm = regexp.MustCompile(`\s+`)
	servicePalette = []string{
		"\x1b[38;5;39m", "\x1b[38;5;202m", "\x1b[38;5;99m", "\x1b[38;5;34m",
		"\x1b[38;5;161m", "\x1b[38;5;208m", "\x1b[38;5;46m", "\x1b[38;5;33m",
		"\x1b[38;5;197m", "\x1b[38;5;82m", "\x1b[38;5;214m", "\x1b[38;5;45m",
	}
	sensitiveFields = map[string]bool{
		"password": true, "token": true, "secret": true, "api_key": true,
		"apikey": true, "access_token": true, "refresh_token": true, "bearer": true,
		"authorization": true, "auth": true, "credential": true, "credentials": true,
		"private_key": true, "privatekey": true, "ssn": true, "credit_card": true,
		"creditcard": true, "cvv": true, "pin": true, "otp": true,
	}
	bufferPool = sync.Pool{
		New: func() interface{} {
			return new(strings.Builder)
		},
	}
)

type zapLogger struct {
	logger      *zap.Logger
	consoleMode bool
	service     string
	color       string
	termWidth   int
	mu          sync.RWMutex
}

type PrettyPrinter struct {
	depth     int
	colorize  bool
	sensitive map[string]bool
}

func NewPrettyPrinter(colorize bool) *PrettyPrinter {
	return &PrettyPrinter{
		depth:     0,
		colorize:  colorize,
		sensitive: sensitiveFields,
	}
}

func (pp *PrettyPrinter) Sprint(v interface{}) string {
	return pp.format(v, 0)
}

func (pp *PrettyPrinter) format(v interface{}, depth int) string {
	if depth > maxDepth {
		return pp.colorWrap("<max depth>", ansiGray)
	}

	if v == nil {
		return pp.colorWrap("null", ansiDim)
	}

	val := reflect.ValueOf(v)
	return pp.formatValue(val, depth)
}

func (pp *PrettyPrinter) formatValue(val reflect.Value, depth int) string {
	if !val.IsValid() {
		return pp.colorWrap("null", ansiDim)
	}

	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return pp.colorWrap("null", ansiDim)
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		return pp.formatStruct(val, depth)
	case reflect.Map:
		return pp.formatMap(val, depth)
	case reflect.Slice, reflect.Array:
		return pp.formatSlice(val, depth)
	case reflect.String:
		return pp.formatString(val.String())
	case reflect.Bool:
		return pp.formatBool(val.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return pp.formatInt(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return pp.formatUint(val.Uint())
	case reflect.Float32, reflect.Float64:
		return pp.formatFloat(val.Float())
	case reflect.Chan:
		return pp.colorWrap(fmt.Sprintf("<chan %s>", val.Type().Elem().String()), ansiGray)
	case reflect.Func:
		return pp.colorWrap("<func>", ansiGray)
	case reflect.UnsafePointer:
		return pp.colorWrap("<ptr>", ansiGray)
	default:
		return fmt.Sprintf("%v", val.Interface())
	}
}

func (pp *PrettyPrinter) formatStruct(val reflect.Value, depth int) string {
	typeName := val.Type().String()

	if typeName == "time.Time" {
		t := val.Interface().(time.Time)
		if t.IsZero() {
			return pp.colorWrap("null", ansiDim)
		}
		return pp.formatString(t.Format(time.RFC3339))
	}

	if val.CanInterface() {
		iface := val.Interface()
		if err, ok := iface.(error); ok {
			return pp.colorWrap(fmt.Sprintf("\"%s\"", err.Error()), ansiRed)
		}
		if stringer, ok := iface.(fmt.Stringer); ok {
			return pp.formatString(stringer.String())
		}
	}

	typ := val.Type()
	numFields := val.NumField()
	if numFields == 0 {
		return "{}"
	}

	type fieldInfo struct {
		name  string
		value reflect.Value
	}
	var fields []fieldInfo

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
			for _, opt := range parts[1:] {
				if opt == "omitempty" && isZeroValue(fieldVal) {
					name = ""
					break
				}
			}
			if name == "" {
				continue
			}
		}

		if isZeroValue(fieldVal) {
			continue
		}

		fields = append(fields, fieldInfo{name: name, value: fieldVal})
	}

	if len(fields) == 0 {
		return "{}"
	}

	sb := getBuffer()
	defer putBuffer(sb)

	indent := strings.Repeat(jsonIndent, depth)
	fieldIndent := strings.Repeat(jsonIndent, depth+1)

	sb.WriteString("{\n")

	for i, f := range fields {
		sb.WriteString(fieldIndent)
		sb.WriteString(pp.colorWrap(fmt.Sprintf("\"%s\"", f.name), ansiCyan))
		sb.WriteString(": ")

		if pp.isSensitive(f.name) {
			sb.WriteString(pp.colorWrap(fmt.Sprintf("\"%s\"", sanitizationMask), ansiYellow))
		} else {
			sb.WriteString(pp.formatValue(f.value, depth+1))
		}

		if i < len(fields)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(indent)
	sb.WriteString("}")

	return sb.String()
}

func (pp *PrettyPrinter) formatMap(val reflect.Value, depth int) string {
	if val.IsNil() {
		return pp.colorWrap("null", ansiDim)
	}

	keys := val.MapKeys()
	if len(keys) == 0 {
		return "{}"
	}

	sb := getBuffer()
	defer putBuffer(sb)

	indent := strings.Repeat(jsonIndent, depth)
	fieldIndent := strings.Repeat(jsonIndent, depth+1)

	sb.WriteString("{\n")

	count := len(keys)
	truncated := false
	if count > maxMapKeys {
		count = maxMapKeys
		truncated = true
	}

	for i := 0; i < count; i++ {
		k := keys[i]
		v := val.MapIndex(k)

		sb.WriteString(fieldIndent)

		keyStr := fmt.Sprintf("%v", k.Interface())
		sb.WriteString(pp.colorWrap(fmt.Sprintf("\"%s\"", keyStr), ansiCyan))
		sb.WriteString(": ")

		if pp.isSensitive(keyStr) {
			sb.WriteString(pp.colorWrap(fmt.Sprintf("\"%s\"", sanitizationMask), ansiYellow))
		} else {
			sb.WriteString(pp.formatValue(v, depth+1))
		}

		if i < count-1 || truncated {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	if truncated {
		sb.WriteString(fieldIndent)
		sb.WriteString(pp.colorWrap(fmt.Sprintf("// ... %d more keys", len(keys)-maxMapKeys), ansiGray))
		sb.WriteString("\n")
	}

	sb.WriteString(indent)
	sb.WriteString("}")

	return sb.String()
}

func (pp *PrettyPrinter) formatSlice(val reflect.Value, depth int) string {
	if val.Kind() == reflect.Slice && val.IsNil() {
		return pp.colorWrap("null", ansiDim)
	}

	length := val.Len()
	if length == 0 {
		return "[]"
	}

	if val.Type().Elem().Kind() == reflect.Uint8 {
		return pp.colorWrap(fmt.Sprintf("<bytes[%d]>", length), ansiGray)
	}

	sb := getBuffer()
	defer putBuffer(sb)

	indent := strings.Repeat(jsonIndent, depth)
	itemIndent := strings.Repeat(jsonIndent, depth+1)

	sb.WriteString("[\n")

	count := length
	truncated := false
	if count > maxArrayDisplay {
		count = maxArrayDisplay
		truncated = true
	}

	for i := 0; i < count; i++ {
		sb.WriteString(itemIndent)
		sb.WriteString(pp.formatValue(val.Index(i), depth+1))

		if i < count-1 || truncated {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	if truncated {
		sb.WriteString(itemIndent)
		sb.WriteString(pp.colorWrap(fmt.Sprintf("// ... %d more items", length-maxArrayDisplay), ansiGray))
		sb.WriteString("\n")
	}

	sb.WriteString(indent)
	sb.WriteString("]")

	return sb.String()
}

func (pp *PrettyPrinter) formatString(s string) string {
	if len(s) > maxStringDisplay {
		s = s[:maxStringDisplay-3] + "..."
	}
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return pp.colorWrap(fmt.Sprintf("\"%s\"", s), ansiGreen)
}

func (pp *PrettyPrinter) formatBool(b bool) string {
	return pp.colorWrap(fmt.Sprintf("%t", b), ansiMagenta)
}

func (pp *PrettyPrinter) formatInt(n int64) string {
	return pp.colorWrap(fmt.Sprintf("%d", n), ansiYellow)
}

func (pp *PrettyPrinter) formatUint(n uint64) string {
	return pp.colorWrap(fmt.Sprintf("%d", n), ansiYellow)
}

func (pp *PrettyPrinter) formatFloat(f float64) string {
	var s string
	if f == float64(int64(f)) {
		s = fmt.Sprintf("%.1f", f)
	} else {
		s = fmt.Sprintf("%g", f)
	}
	return pp.colorWrap(s, ansiYellow)
}

func (pp *PrettyPrinter) colorWrap(s, color string) string {
	if !pp.colorize {
		return s
	}
	return color + s + ansiReset
}

func (pp *PrettyPrinter) isSensitive(key string) bool {
	lower := strings.ToLower(key)
	if pp.sensitive[lower] {
		return true
	}
	for word := range pp.sensitive {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}

func PrettyJSON(v interface{}) string {
	return NewPrettyPrinter(true).Sprint(v)
}

func PrettyJSONNoColor(v interface{}) string {
	return NewPrettyPrinter(false).Sprint(v)
}

func getBuffer() *strings.Builder {
	sb := bufferPool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

func putBuffer(sb *strings.Builder) {
	bufferPool.Put(sb)
}

func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < minTerminalWidth {
		return defaultTerminalWidth
	}
	return width
}

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func visibleLen(s string) int {
	return utf8.RuneCountInString(stripANSI(s))
}

func padANSI(s string, target int) string {
	if target <= 0 {
		return s
	}
	if !ansiPattern.MatchString(s) {
		return padPlain(s, target)
	}
	visible := visibleLen(s)
	if visible == target {
		return s
	}
	if visible < target {
		return s + strings.Repeat(" ", target-visible)
	}
	return truncateWithANSI(s, target)
}

func padPlain(s string, target int) string {
	if target < 0 {
		target = 0
	}
	visible := utf8.RuneCountInString(s)
	if visible == target {
		return s
	}
	if visible < target {
		return s + strings.Repeat(" ", target-visible)
	}
	if target > truncationMinWidth {
		runes := []rune(s)
		truncateAt := target - truncationMinWidth
		if truncateAt > len(runes) {
			truncateAt = len(runes)
		}
		return string(runes[:truncateAt]) + truncationSuffix
	}
	if target <= 0 {
		return ""
	}
	runes := []rune(s)
	if target > len(runes) {
		target = len(runes)
	}
	return string(runes[:target])
}

func truncateWithANSI(s string, target int) string {
	sb := getBuffer()
	defer putBuffer(sb)

	visibleCount := 0
	i := 0
	bs := []byte(s)
	targetVisible := target - truncationMinWidth

	for i < len(bs) && visibleCount < targetVisible {
		if bs[i] == 0x1b && i+1 < len(bs) && bs[i+1] == '[' {
			j := i + 2
			for j < len(bs) && ((bs[j] >= '0' && bs[j] <= '9') || bs[j] == ';') {
				j++
			}
			if j < len(bs) && bs[j] == 'm' {
				sb.Write(bs[i : j+1])
				i = j + 1
				continue
			}
		}
		r, size := utf8.DecodeRune(bs[i:])
		sb.WriteRune(r)
		visibleCount++
		i += size
	}

	sb.WriteString(ansiReset + truncationSuffix)
	res := sb.String()
	resVisible := visibleLen(res)
	if resVisible < target {
		res += strings.Repeat(" ", target-resVisible)
	}
	return res
}

func truncateStart(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max > truncationMinWidth {
		return truncationSuffix + s[len(s)-(max-truncationMinWidth):]
	}
	return s[len(s)-max:]
}

func truncateEnd(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max > truncationMinWidth {
		return s[:max-truncationMinWidth] + truncationSuffix
	}
	return s[:max]
}

func customStackTrace(skip int, pad int) string {
	sb := getBuffer()
	defer putBuffer(sb)

	prefix := strings.Repeat(" ", pad)

	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		funcName := "unknown"
		if fn != nil {
			funcName = fn.Name()
			if lastSlash := strings.LastIndex(funcName, "/"); lastSlash != -1 {
				funcName = funcName[lastSlash+1:]
			}
		}

		shortFile := file
		if idx := strings.LastIndex(file, "/"); idx != -1 {
			shortFile = file[idx+1:]
		}

		sb.WriteString(fmt.Sprintf("%s%s%s%s -> %s%s:%d%s\n",
			prefix, ansiDim, funcName, ansiReset,
			ansiCyan, shortFile, line, ansiReset))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			result = append(result, "")
			continue
		}

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "{") ||
			strings.HasPrefix(trimmed, "[") ||
			strings.HasPrefix(trimmed, "}") ||
			strings.HasPrefix(trimmed, "]") ||
			strings.Contains(line, "\": ") ||
			strings.HasPrefix(trimmed, "//") {
			result = append(result, line)
			continue
		}

		normalized := whitespaceNorm.ReplaceAllString(trimmed, " ")

		if visibleLen(normalized) <= width {
			result = append(result, normalized)
			continue
		}

		current := normalized
		for visibleLen(current) > width {
			breakPoint := width
			stripped := stripANSI(current)
			if len(stripped) > width {
				lastSpace := strings.LastIndex(stripped[:width], " ")
				if lastSpace > width/2 {
					breakPoint = lastSpace
				}
			}
			result = append(result, current[:breakPoint])
			current = strings.TrimSpace(current[breakPoint:])
		}
		if len(current) > 0 {
			result = append(result, current)
		}
	}
	return result
}

func calculateRequestBoxDimensions(termWidth int) BoxDimensions {
	if termWidth < minTerminalWidth {
		termWidth = defaultTerminalWidth
	}

	dims := BoxDimensions{Width: termWidth}
	dims.Timestamp = ColumnConfig{Start: boxBorderWidth, Width: timestampColumnWidth}
	dims.Level = ColumnConfig{Start: dims.Timestamp.Start + dims.Timestamp.Width + columnSeparatorWidth, Width: levelColumnWidth}
	dims.File = ColumnConfig{Start: dims.Level.Start + dims.Level.Width + columnSeparatorWidth, Width: fileColumnWidth}
	dims.Service = ColumnConfig{Start: dims.File.Start + dims.File.Width + columnSeparatorWidth, Width: serviceColumnWidth}
	dims.Method = ColumnConfig{Start: dims.Service.Start + dims.Service.Width + columnSeparatorWidth, Width: methodColumnWidth}
	dims.Status = ColumnConfig{Start: dims.Method.Start + dims.Method.Width + columnSeparatorWidth, Width: statusColumnWidth}
	dims.Duration = ColumnConfig{Start: dims.Status.Start + dims.Status.Width + columnSeparatorWidth, Width: durationColumnWidth}
	dims.BodySize = ColumnConfig{Start: dims.Duration.Start + dims.Duration.Width + columnSeparatorWidth, Width: bodySizeColumnWidth}
	dims.RoutePath = ColumnConfig{Start: dims.BodySize.Start + dims.BodySize.Width + columnSeparatorWidth, Width: routePathColumnWidth}
	dims.Message = ColumnConfig{Start: dims.RoutePath.Start + dims.RoutePath.Width + columnSeparatorWidth, Width: termWidth - dims.RoutePath.Start - dims.RoutePath.Width - columnSeparatorWidth - boxBorderWidth}
	return dims
}

func calculateStandardBoxDimensions(termWidth int) BoxDimensions {
	if termWidth < minTerminalWidth {
		termWidth = defaultTerminalWidth
	}

	dims := BoxDimensions{Width: termWidth}
	dims.Timestamp = ColumnConfig{Start: boxBorderWidth, Width: timestampColumnWidth}
	dims.Level = ColumnConfig{Start: dims.Timestamp.Start + dims.Timestamp.Width + columnSeparatorWidth, Width: levelColumnWidth}
	dims.File = ColumnConfig{Start: dims.Level.Start + dims.Level.Width + columnSeparatorWidth, Width: fileColumnWidth}
	dims.Service = ColumnConfig{Start: dims.File.Start + dims.File.Width + columnSeparatorWidth, Width: serviceColumnWidth}
	dims.Message = ColumnConfig{Start: dims.Service.Start + dims.Service.Width + columnSeparatorWidth, Width: termWidth - dims.Service.Start - dims.Service.Width - columnSeparatorWidth - boxBorderWidth}
	return dims
}

func buildBorderWithSeparators(width int, separatorPositions []int, topBorder bool) string {
	var leftChar, rightChar, joinChar string
	if topBorder {
		leftChar, rightChar, joinChar = "┌", "┐", "┬"
	} else {
		leftChar, rightChar, joinChar = "└", "┘", "┴"
	}

	sb := getBuffer()
	defer putBuffer(sb)

	sb.WriteString(leftChar)

	sepSet := make(map[int]bool, len(separatorPositions))
	for _, pos := range separatorPositions {
		sepSet[pos] = true
	}

	for i := 1; i < width-1; i++ {
		if sepSet[i+1] {
			sb.WriteString(joinChar)
		} else {
			sb.WriteString("─")
		}
	}

	sb.WriteString(rightChar)
	sb.WriteString("\n")
	return sb.String()
}

func getLevelColor(level string) string {
	switch level {
	case "DEBUG":
		return ansiBrightCyan
	case "INFO":
		return ansiGreen
	case "WARN":
		return ansiYellow
	case "ERROR", "FATAL":
		return ansiRed
	default:
		return ansiWhite
	}
}

func getStatusColor(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return ansiGreen
	case statusCode >= 300 && statusCode < 400:
		return ansiCyan
	case statusCode >= 400 && statusCode < 500:
		return ansiYellow
	case statusCode >= 500:
		return ansiRed
	default:
		return ansiWhite
	}
}

func formatBodySize(size int64) string {
	if size < 0 {
		return "0B"
	}
	if size < bytesPerKilobyte {
		return fmt.Sprintf("%dB", size)
	}
	div, exp := int64(bytesPerKilobyte), 0
	for n := size / bytesPerKilobyte; n >= bytesPerKilobyte; n /= bytesPerKilobyte {
		div *= bytesPerKilobyte
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
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

func isSensitiveField(key string) bool {
	lower := strings.ToLower(key)
	if sensitiveFields[lower] {
		return true
	}
	for word := range sensitiveFields {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}

func sanitizeValue(value string) string {
	if len(value) <= sensitiveFieldPrefix {
		return sanitizationMask
	}
	return value[:sensitiveFieldPrefix] + sanitizationMask
}

func (l *zapLogger) drawRequestBox(timestamp, level, file, service, method, routePath string, statusCode int, duration time.Duration, bodySize int64, message string) string {
	sb := getBuffer()
	defer putBuffer(sb)

	dims := calculateRequestBoxDimensions(l.termWidth)
	separatorPositions := []int{
		dims.Level.Start - 1, dims.File.Start - 1, dims.Service.Start - 1,
		dims.Method.Start - 1, dims.Status.Start - 1, dims.Duration.Start - 1,
		dims.BodySize.Start - 1, dims.RoutePath.Start - 1, dims.Message.Start - 1,
	}

	sb.WriteString(buildBorderWithSeparators(dims.Width, separatorPositions, true))

	durationStr := formatDuration(duration)
	bodySizeStr := formatBodySize(bodySize)
	coloredStatus := fmt.Sprintf("%s%d%s", getStatusColor(statusCode), statusCode, ansiReset)

	messageLines := wrapText(message, dims.Message.Width)
	msgPart := ""
	if len(messageLines) > 0 {
		msgPart = messageLines[0]
	}

	line := fmt.Sprintf("│ %s │ %s │ %s │ %s │ %s │ %s │ %s │ %s │ %s │ %s",
		padPlain(timestamp, dims.Timestamp.Width),
		padANSI(fmt.Sprintf("%s%s%s", getLevelColor(level), level, ansiReset), dims.Level.Width),
		padPlain(file, dims.File.Width),
		padANSI(service, dims.Service.Width),
		padPlain(method, dims.Method.Width),
		padANSI(coloredStatus, dims.Status.Width),
		padPlain(durationStr, dims.Duration.Width),
		padPlain(bodySizeStr, dims.BodySize.Width),
		padPlain(routePath, dims.RoutePath.Width),
		padPlain(msgPart, dims.Message.Width))

	visible := visibleLen(line)
	padding := dims.Width - visible - boxBorderWidth
	if padding < 0 {
		padding = 0
	}
	sb.WriteString(line + strings.Repeat(" ", padding) + " │\n")

	for i := 1; i < len(messageLines); i++ {
		contLine := fmt.Sprintf("│ %s │ %s │ %s │ %s │ %s │ %s │ %s │ %s │ %s │ %s",
			strings.Repeat(" ", dims.Timestamp.Width),
			strings.Repeat(" ", dims.Level.Width),
			strings.Repeat(" ", dims.File.Width),
			strings.Repeat(" ", dims.Service.Width),
			strings.Repeat(" ", dims.Method.Width),
			strings.Repeat(" ", dims.Status.Width),
			strings.Repeat(" ", dims.Duration.Width),
			strings.Repeat(" ", dims.BodySize.Width),
			strings.Repeat(" ", dims.RoutePath.Width),
			padPlain(messageLines[i], dims.Message.Width))

		visible := visibleLen(contLine)
		padding := dims.Width - visible - boxBorderWidth
		if padding < 0 {
			padding = 0
		}
		sb.WriteString(contLine + strings.Repeat(" ", padding) + " │\n")
	}

	sb.WriteString(buildBorderWithSeparators(dims.Width, separatorPositions, false))
	return sb.String()
}

func (l *zapLogger) drawBoxedLog(timestamp, level, file, service, message string) string {
	sb := getBuffer()
	defer putBuffer(sb)

	dims := calculateStandardBoxDimensions(l.termWidth)
	separatorPositions := []int{
		dims.Level.Start - 1, dims.File.Start - 1,
		dims.Service.Start - 1, dims.Message.Start - 1,
	}

	sb.WriteString(buildBorderWithSeparators(dims.Width, separatorPositions, true))

	messageLines := wrapText(message, dims.Message.Width)
	msgPart := ""
	if len(messageLines) > 0 {
		msgPart = messageLines[0]
	}

	line := fmt.Sprintf("│ %s │ %s │ %s │ %s │ %s",
		padPlain(timestamp, dims.Timestamp.Width),
		padANSI(fmt.Sprintf("%s%s%s", getLevelColor(level), level, ansiReset), dims.Level.Width),
		padPlain(file, dims.File.Width),
		padANSI(service, dims.Service.Width),
		padPlain(msgPart, dims.Message.Width))

	visible := visibleLen(line)
	padding := dims.Width - visible - boxBorderWidth
	if padding < 0 {
		padding = 0
	}
	sb.WriteString(line + strings.Repeat(" ", padding) + " │\n")

	for i := 1; i < len(messageLines); i++ {
		contLine := fmt.Sprintf("│ %s │ %s │ %s │ %s │ %s",
			strings.Repeat(" ", dims.Timestamp.Width),
			strings.Repeat(" ", dims.Level.Width),
			strings.Repeat(" ", dims.File.Width),
			strings.Repeat(" ", dims.Service.Width),
			padPlain(messageLines[i], dims.Message.Width))

		visible := visibleLen(contLine)
		padding := dims.Width - visible - boxBorderWidth
		if padding < 0 {
			padding = 0
		}
		sb.WriteString(contLine + strings.Repeat(" ", padding) + " │\n")
	}

	sb.WriteString(buildBorderWithSeparators(dims.Width, separatorPositions, false))
	return sb.String()
}

func truncateCaller(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("")
}

func padLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("")
}

func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("")
}

func pickColor(service string) string {
	h := fnv.New32a()
	h.Write([]byte(service))
	return servicePalette[int(h.Sum32())%len(servicePalette)]
}

func toZapLevel(level logger.Level) zapcore.Level {
	switch level {
	case logger.DebugLevel:
		return zapcore.DebugLevel
	case logger.InfoLevel:
		return zapcore.InfoLevel
	case logger.WarnLevel:
		return zapcore.WarnLevel
	case logger.ErrorLevel:
		return zapcore.ErrorLevel
	case logger.FatalLevel:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func NewZap(cfg logger.Config) (logger.Logger, error) {
	var zapCfg zap.Config

	if cfg.Format == logger.FormatText {
		zapCfg = zap.Config{
			Level:            zap.NewAtomicLevelAt(toZapLevel(cfg.Level)),
			Development:      true,
			Encoding:         "console",
			EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
		}
		zapCfg.EncoderConfig.EncodeLevel = padLevelEncoder
		zapCfg.EncoderConfig.EncodeTime = customTimeEncoder
		zapCfg.EncoderConfig.EncodeCaller = truncateCaller
		zapCfg.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
		zapCfg.EncoderConfig.ConsoleSeparator = ""
		zapCfg.EncoderConfig.LineEnding = "\n"
	} else {
		zapCfg = zap.NewProductionConfig()
		zapCfg.EncoderConfig.TimeKey = "timestamp"
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapCfg.EncoderConfig.CallerKey = "caller"
		zapCfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
		zapCfg.EncoderConfig.MessageKey = "message"
		zapCfg.EncoderConfig.LevelKey = "level"
		zapCfg.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		zapCfg.Encoding = "json"
	}

	zapCfg.Level = zap.NewAtomicLevelAt(toZapLevel(cfg.Level))
	zapCfg.OutputPaths = []string{"stdout"}
	zapCfg.ErrorOutputPaths = []string{"stderr"}
	zapCfg.DisableCaller = false
	zapCfg.DisableStacktrace = true

	zl, err := zapCfg.Build(zap.AddCallerSkip(callerSkipDefault))
	if err != nil {
		return nil, err
	}

	return &zapLogger{
		logger:      zl,
		consoleMode: cfg.Format == logger.FormatText,
		service:     cfg.Service,
		color:       pickColor(cfg.Service),
		termWidth:   getTerminalWidth(),
	}, nil
}

func (l *zapLogger) makeZapFields(extra []logger.Field) []zap.Field {
	if len(extra) == 0 {
		return nil
	}

	zfs := make([]zap.Field, 0, len(extra))
	for _, f := range extra {
		if f == nil {
			continue
		}
		k := f.Key()
		v := f.Value()

		if v == nil {
			continue
		}

		if isSensitiveField(k) {
			switch val := v.(type) {
			case string:
				zfs = append(zfs, zap.String(k, sanitizeValue(val)))
			default:
				zfs = append(zfs, zap.String(k, sanitizeValue(fmt.Sprintf("%v", val))))
			}
			continue
		}

		switch val := v.(type) {
		case string:
			zfs = append(zfs, zap.String(k, val))
		case []string:
			zfs = append(zfs, zap.Strings(k, val))
		case int:
			zfs = append(zfs, zap.Int(k, val))
		case int8:
			zfs = append(zfs, zap.Int8(k, val))
		case int16:
			zfs = append(zfs, zap.Int16(k, val))
		case int32:
			zfs = append(zfs, zap.Int32(k, val))
		case int64:
			zfs = append(zfs, zap.Int64(k, val))
		case uint:
			zfs = append(zfs, zap.Uint(k, val))
		case uint8:
			zfs = append(zfs, zap.Uint8(k, val))
		case uint16:
			zfs = append(zfs, zap.Uint16(k, val))
		case uint32:
			zfs = append(zfs, zap.Uint32(k, val))
		case uint64:
			zfs = append(zfs, zap.Uint64(k, val))
		case bool:
			zfs = append(zfs, zap.Bool(k, val))
		case float32:
			zfs = append(zfs, zap.Float32(k, val))
		case float64:
			zfs = append(zfs, zap.Float64(k, val))
		case time.Time:
			zfs = append(zfs, zap.Time(k, val))
		case time.Duration:
			zfs = append(zfs, zap.Duration(k, val))
		case error:
			if val != nil {
				zfs = append(zfs, zap.String(k, val.Error()))
			}
		case []byte:
			zfs = append(zfs, zap.Binary(k, val))
		default:
			rv := reflect.ValueOf(val)
			switch rv.Kind() {
			case reflect.Struct, reflect.Ptr, reflect.Slice, reflect.Array, reflect.Map:
				jsonBytes, err := json.Marshal(val)
				if err != nil {
					zfs = append(zfs, zap.Any(k, val))
				} else {
					zfs = append(zfs, zap.String(k, string(jsonBytes)))
				}
			default:
				zfs = append(zfs, zap.Any(k, val))
			}
		}
	}
	return zfs
}

func formatFieldsPretty(fields []logger.Field) []string {
	if len(fields) == 0 {
		return nil
	}

	pp := NewPrettyPrinter(true)
	var lines []string

	for _, f := range fields {
		if f == nil {
			continue
		}

		k := f.Key()
		v := f.Value()

		if v == nil {
			lines = append(lines, fmt.Sprintf("  %s%s%s: %snull%s", ansiBold+ansiCyan, k, ansiReset, ansiDim, ansiReset))
			continue
		}

		if isSensitiveField(k) {
			lines = append(lines, fmt.Sprintf("  %s%s%s: %s\"%s\"%s", ansiBold+ansiCyan, k, ansiReset, ansiYellow, sanitizationMask, ansiReset))
			continue
		}

		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				lines = append(lines, fmt.Sprintf("  %s%s%s: %snull%s", ansiBold+ansiCyan, k, ansiReset, ansiDim, ansiReset))
				continue
			}
			rv = rv.Elem()
			v = rv.Interface()
		}

		switch rv.Kind() {
		case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array:
			formatted := pp.Sprint(v)
			formattedLines := strings.Split(formatted, "\n")

			if len(formattedLines) == 1 {
				lines = append(lines, fmt.Sprintf("  %s%s%s: %s", ansiBold+ansiCyan, k, ansiReset, formatted))
			} else {
				lines = append(lines, fmt.Sprintf("  %s%s%s:", ansiBold+ansiCyan, k, ansiReset))
				for _, fl := range formattedLines {
					lines = append(lines, "    "+fl)
				}
			}

		case reflect.String:
			s := rv.String()
			if len(s) > maxStringDisplay {
				s = s[:maxStringDisplay-3] + "..."
			}
			lines = append(lines, fmt.Sprintf("  %s%s%s: %s\"%s\"%s", ansiBold+ansiCyan, k, ansiReset, ansiGreen, s, ansiReset))

		case reflect.Bool:
			lines = append(lines, fmt.Sprintf("  %s%s%s: %s%t%s", ansiBold+ansiCyan, k, ansiReset, ansiMagenta, rv.Bool(), ansiReset))

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			lines = append(lines, fmt.Sprintf("  %s%s%s: %s%d%s", ansiBold+ansiCyan, k, ansiReset, ansiYellow, rv.Int(), ansiReset))

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			lines = append(lines, fmt.Sprintf("  %s%s%s: %s%d%s", ansiBold+ansiCyan, k, ansiReset, ansiYellow, rv.Uint(), ansiReset))

		case reflect.Float32, reflect.Float64:
			fv := rv.Float()
			if fv == float64(int64(fv)) {
				lines = append(lines, fmt.Sprintf("  %s%s%s: %s%.1f%s", ansiBold+ansiCyan, k, ansiReset, ansiYellow, fv, ansiReset))
			} else {
				lines = append(lines, fmt.Sprintf("  %s%s%s: %s%g%s", ansiBold+ansiCyan, k, ansiReset, ansiYellow, fv, ansiReset))
			}

		case reflect.Chan:
			lines = append(lines, fmt.Sprintf("  %s%s%s: %s<chan %s>%s", ansiBold+ansiCyan, k, ansiReset, ansiGray, rv.Type().Elem().String(), ansiReset))

		case reflect.Func:
			lines = append(lines, fmt.Sprintf("  %s%s%s: %s<func>%s", ansiBold+ansiCyan, k, ansiReset, ansiGray, ansiReset))

		default:
			lines = append(lines, fmt.Sprintf("  %s%s%s: %v", ansiBold+ansiCyan, k, ansiReset, v))
		}
	}

	return lines
}

func (l *zapLogger) formatLog(level string, msg string, fields []logger.Field) string {
	_, file, line, _ := runtime.Caller(callerSkipFormatLog)

	parts := strings.Split(file, "/")
	shortFile := parts[len(parts)-1]
	fileLoc := fmt.Sprintf("%s:%d", shortFile, line)
	if len(fileLoc) > fileColumnWidth {
		fileLoc = truncateStart(fileLoc, fileColumnWidth)
	}

	service := truncateEnd(l.service, serviceColumnWidth)
	if l.color != "" {
		service = fmt.Sprintf("%s%s%s", l.color, service, ansiReset)
	}
	service = padANSI(service, serviceColumnWidth)

	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05.000")

	message := msg
	fieldLines := formatFieldsPretty(fields)
	if len(fieldLines) > 0 {
		message = msg + "\n" + strings.Join(fieldLines, "\n")
	}

	return l.drawBoxedLog(timestamp, level, fileLoc, service, message)
}

func (l *zapLogger) Debug(msg string, fields ...logger.Field) {
	l.mu.RLock()
	consoleMode := l.consoleMode
	l.mu.RUnlock()

	if consoleMode && l.logger.Core().Enabled(zapcore.DebugLevel) {
		fmt.Print(l.formatLog("DEBUG", msg, fields))
		return
	}
	l.logger.Debug(msg, l.makeZapFields(fields)...)
}

func (l *zapLogger) Info(msg string, fields ...logger.Field) {
	l.mu.RLock()
	consoleMode := l.consoleMode
	l.mu.RUnlock()

	if consoleMode && l.logger.Core().Enabled(zapcore.InfoLevel) {
		fmt.Print(l.formatLog("INFO", msg, fields))
		return
	}
	l.logger.Info(msg, l.makeZapFields(fields)...)
}

func (l *zapLogger) Warn(msg string, fields ...logger.Field) {
	l.mu.RLock()
	consoleMode := l.consoleMode
	l.mu.RUnlock()

	if consoleMode && l.logger.Core().Enabled(zapcore.WarnLevel) {
		fmt.Print(l.formatLog("WARN", msg, fields))
		return
	}
	l.logger.Warn(msg, l.makeZapFields(fields)...)
}

func (l *zapLogger) Error(msg string, fields ...logger.Field) {
	l.mu.RLock()
	consoleMode := l.consoleMode
	l.mu.RUnlock()

	if consoleMode && l.logger.Core().Enabled(zapcore.ErrorLevel) {
		content := msg + "\n" + customStackTrace(callerSkipError, 0)
		fmt.Print(l.formatLog("ERROR", content, fields))
		return
	}
	zfs := l.makeZapFields(fields)
	zfs = append(zfs, zap.String("stack", customStackTrace(callerSkipError, 0)))
	l.logger.Error(msg, zfs...)
}

func (l *zapLogger) Fatal(msg string, fields ...logger.Field) {
	l.mu.RLock()
	consoleMode := l.consoleMode
	l.mu.RUnlock()

	if consoleMode {
		content := msg + "\n" + customStackTrace(callerSkipError, 0)
		fmt.Print(l.formatLog("FATAL", content, fields))
		os.Exit(1)
		return
	}
	zfs := l.makeZapFields(fields)
	zfs = append(zfs, zap.String("stack", customStackTrace(callerSkipError, 0)))
	l.logger.Fatal(msg, zfs...)
}

func (l *zapLogger) Request(ctx context.Context, method string, routePath string, statusCode int, duration time.Duration, bodySize int64, msg string, fields ...logger.Field) {
	l.mu.RLock()
	consoleMode := l.consoleMode
	l.mu.RUnlock()

	if consoleMode {
		_, file, line, _ := runtime.Caller(callerSkipDefault)

		parts := strings.Split(file, "/")
		shortFile := parts[len(parts)-1]
		fileLoc := fmt.Sprintf("%s:%d", shortFile, line)
		if len(fileLoc) > fileColumnWidth {
			fileLoc = truncateStart(fileLoc, fileColumnWidth)
		}

		service := truncateEnd(l.service, serviceColumnWidth)
		if l.color != "" {
			service = fmt.Sprintf("%s%s%s", l.color, service, ansiReset)
		}
		service = padANSI(service, serviceColumnWidth)

		timestamp := time.Now().UTC().Format("2006-01-02 15:04:05.000")

		message := msg
		if len(fields) > 0 {
			extraFields := make([]logger.Field, 0, len(fields))
			for _, f := range fields {
				if f != nil && f.Key() != "status" {
					extraFields = append(extraFields, f)
				}
			}
			if len(extraFields) > 0 {
				fieldLines := formatFieldsPretty(extraFields)
				if len(fieldLines) > 0 {
					message = msg + "\n" + strings.Join(fieldLines, "\n")
				}
			}
		}

		fmt.Print(l.drawRequestBox(timestamp, "INFO", fileLoc, service, method, routePath, statusCode, duration, bodySize, message))
		return
	}

	zfs := l.makeZapFields(fields)
	zfs = append(zfs,
		zap.String("method", method),
		zap.String("route", routePath),
		zap.Int("status", statusCode),
		zap.Int64("duration_ms", duration.Milliseconds()),
		zap.Int64("body_size", bodySize),
		zap.String("service", l.service),
	)
	l.logger.Info(msg, zfs...)
}

func (l *zapLogger) With(fields ...logger.Field) logger.Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return &zapLogger{
		logger:      l.logger.With(l.makeZapFields(fields)...),
		consoleMode: l.consoleMode,
		service:     l.service,
		color:       l.color,
		termWidth:   l.termWidth,
	}
}

func (l *zapLogger) WithContext(ctx context.Context) logger.Logger {
	return l
}

func (l *zapLogger) Sync() error {
	return l.logger.Sync()
}
