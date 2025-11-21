package adapter

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"regexp"
	"runtime"
	"strings"
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
	defaultTerminalWidth = 200
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
	ansiItalic     = "\x1b[3m"

	bytesPerKilobyte = 1024

	fieldSeparator      = " | "
	fieldKeyValueFormat = "%s=%v"

	sensitiveFieldPrefix = 4
	sanitizationMask     = "****"
)

var (
	ansiPattern    = regexp.MustCompile(`\x1b\[[0-9;]*m`)
	servicePalette = []string{
		"\x1b[38;5;39m", "\x1b[38;5;202m", "\x1b[38;5;99m", "\x1b[38;5;34m",
		"\x1b[38;5;161m", "\x1b[38;5;208m", "\x1b[38;5;46m", "\x1b[38;5;33m",
	}
	sensitiveFields = map[string]bool{
		"password":      true,
		"token":         true,
		"secret":        true,
		"api_key":       true,
		"apikey":        true,
		"access_token":  true,
		"refresh_token": true,
		"bearer":        true,
		"authorization": true,
		"auth":          true,
		"credential":    true,
		"credentials":   true,
		"private_key":   true,
		"privatekey":    true,
	}
)

type zapLogger struct {
	logger      *zap.Logger
	consoleMode bool
	service     string
	color       string
	termWidth   int
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

	// Handle truncation
	if target > truncationMinWidth {
		runes := []rune(s)
		truncateAt := target - truncationMinWidth
		if truncateAt > len(runes) {
			truncateAt = len(runes)
		}
		return string(runes[:truncateAt]) + truncationSuffix
	}

	// If target is too small for truncation suffix, just truncate
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
	var out strings.Builder
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
				out.Write(bs[i : j+1])
				i = j + 1
				continue
			}
		}
		r, size := utf8.DecodeRune(bs[i:])
		out.WriteRune(r)
		visibleCount++
		i += size
	}

	out.WriteString(ansiReset + truncationSuffix)
	res := out.String()
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
	var sb strings.Builder
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
		sb.WriteString(fmt.Sprintf("%s%s -> %s:%d\n", prefix, funcName, file, line))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	// Normalize spaces - replace multiple spaces with single space
	// but preserve intentional line breaks
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		// Normalize spaces in the line
		line = strings.TrimSpace(line)
		line = regexp.MustCompile(`\s+`).ReplaceAllString(line, " ")

		if visibleLen(line) <= width {
			result = append(result, line)
			continue
		}

		for visibleLen(line) > width {
			breakPoint := width
			stripped := stripANSI(line)
			if len(stripped) > width {
				lastSpace := strings.LastIndex(stripped[:width], " ")
				halfWidth := width / 2
				if lastSpace > halfWidth {
					breakPoint = lastSpace
				}
			}

			result = append(result, line[:breakPoint])
			line = strings.TrimSpace(line[breakPoint:])
		}
		if len(line) > 0 {
			result = append(result, line)
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

	border := leftChar
	for i := 1; i < width-1; i++ {
		pos := i + 1
		isSeparator := false
		for _, sepPos := range separatorPositions {
			if pos == sepPos {
				isSeparator = true
				break
			}
		}
		if isSeparator {
			border += joinChar
		} else {
			border += "─"
		}
	}
	border += rightChar + "\n"
	return border
}

func getLevelColor(level string) string {
	switch level {
	case "DEBUG":
		return ansiBrightCyan
	case "INFO":
		return ansiGreen
	case "WARN":
		return ansiYellow
	case "ERROR":
		return ansiRed
	case "FATAL":
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
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	} else if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

func isSensitiveField(key string) bool {
	lowerKey := strings.ToLower(key)

	if sensitiveFields[lowerKey] {
		return true
	}

	for sensitiveWord := range sensitiveFields {
		if strings.Contains(lowerKey, sensitiveWord) {
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
	var result strings.Builder

	dims := calculateRequestBoxDimensions(l.termWidth)

	separatorPositions := []int{
		dims.Level.Start - 1,
		dims.File.Start - 1,
		dims.Service.Start - 1,
		dims.Method.Start - 1,
		dims.Status.Start - 1,
		dims.Duration.Start - 1,
		dims.BodySize.Start - 1,
		dims.RoutePath.Start - 1,
		dims.Message.Start - 1,
	}

	result.WriteString(buildBorderWithSeparators(dims.Width, separatorPositions, true))

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
	result.WriteString(line + strings.Repeat(" ", padding) + " │\n")

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
		result.WriteString(contLine + strings.Repeat(" ", padding) + " │\n")
	}

	result.WriteString(buildBorderWithSeparators(dims.Width, separatorPositions, false))
	return result.String()
}

func (l *zapLogger) drawBoxedLog(timestamp, level, file, service, message string) string {
	var result strings.Builder

	dims := calculateStandardBoxDimensions(l.termWidth)

	separatorPositions := []int{
		dims.Level.Start - 1,
		dims.File.Start - 1,
		dims.Service.Start - 1,
		dims.Message.Start - 1,
	}

	result.WriteString(buildBorderWithSeparators(dims.Width, separatorPositions, true))

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
	result.WriteString(line + strings.Repeat(" ", padding) + " │\n")

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
		result.WriteString(contLine + strings.Repeat(" ", padding) + " │\n")
	}

	result.WriteString(buildBorderWithSeparators(dims.Width, separatorPositions, false))
	return result.String()
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
	zfs := make([]zap.Field, 0, len(extra))
	for _, f := range extra {
		if f == nil {
			continue
		}
		k := f.Key()

		if isSensitiveField(k) {
			switch v := f.Value().(type) {
			case string:
				zfs = append(zfs, zap.String(k, sanitizeValue(v)))
			default:
				strVal := fmt.Sprintf("%v", v)
				zfs = append(zfs, zap.String(k, sanitizeValue(strVal)))
			}
			continue
		}

		switch v := f.Value().(type) {
		case string:
			zfs = append(zfs, zap.String(k, v))
		case int:
			zfs = append(zfs, zap.Int(k, v))
		case int64:
			zfs = append(zfs, zap.Int64(k, v))
		case bool:
			zfs = append(zfs, zap.Bool(k, v))
		case float64:
			zfs = append(zfs, zap.Float64(k, v))
		case error:
			zfs = append(zfs, zap.Error(v))
		default:
			zfs = append(zfs, zap.Any(k, v))
		}
	}
	return zfs
}

func formatFields(fields []logger.Field, maxWidth int) []string {
	if len(fields) == 0 {
		return []string{}
	}

	// Build formatted field strings with ANSI formatting
	fieldStrs := make([]string, 0, len(fields))
	for _, f := range fields {
		if f == nil {
			continue
		}
		k := f.Key()
		v := f.Value()

		if isSensitiveField(k) {
			switch val := v.(type) {
			case string:
				v = sanitizeValue(val)
			default:
				v = sanitizeValue(fmt.Sprintf("%v", val))
			}
		}

		if k == "error" {
			// For error fields, make the entire error message italic
			fieldStrs = append(fieldStrs, fmt.Sprintf("* %s%s%s", ansiItalic, v, ansiReset))
		} else {
			// Bold key, italic value
			fieldStrs = append(fieldStrs, fmt.Sprintf("* %s%s%s: %s%v%s", ansiBold, k, ansiReset, ansiItalic, v, ansiReset))
		}
	}

	if len(fieldStrs) == 0 {
		return []string{}
	}

	// Intelligently group fields into lines based on their visible lengths (without ANSI codes)
	var lines []string
	var currentLine []string
	currentLineLen := 0

	for _, field := range fieldStrs {
		// Get visible length (without ANSI codes)
		fieldLen := visibleLen(field)

		// Check if adding this field would exceed the width
		// Account for spacing between fields (3 spaces minimum)
		neededSpace := fieldLen
		if len(currentLine) > 0 {
			neededSpace += 3 // minimum spacing between fields
		}

		// If this field alone is longer than maxWidth, put it on its own line
		if fieldLen > maxWidth {
			// Flush current line if it has content
			if len(currentLine) > 0 {
				lines = append(lines, formatFieldLine(currentLine, maxWidth))
				currentLine = []string{}
				currentLineLen = 0
			}
			// Add the long field on its own line
			lines = append(lines, field)
			continue
		}

		// If adding this field would exceed width, start a new line
		if currentLineLen+neededSpace > maxWidth {
			// Flush current line
			if len(currentLine) > 0 {
				lines = append(lines, formatFieldLine(currentLine, maxWidth))
			}
			// Start new line with this field
			currentLine = []string{field}
			currentLineLen = fieldLen
		} else {
			// Add field to current line
			currentLine = append(currentLine, field)
			currentLineLen += neededSpace
		}

		// Maximum 3 fields per line for readability
		if len(currentLine) >= 3 {
			lines = append(lines, formatFieldLine(currentLine, maxWidth))
			currentLine = []string{}
			currentLineLen = 0
		}
	}

	// Add any remaining fields
	if len(currentLine) > 0 {
		lines = append(lines, formatFieldLine(currentLine, maxWidth))
	}

	return lines
}

// formatFieldLine formats a line of fields with equal spacing
func formatFieldLine(fields []string, maxWidth int) string {
	if len(fields) == 0 {
		return ""
	}

	if len(fields) == 1 {
		return fields[0]
	}

	// Calculate total visible length of all fields (excluding ANSI codes)
	totalFieldLen := 0
	for _, f := range fields {
		totalFieldLen += visibleLen(f)
	}

	// Calculate available space for padding between fields
	availableSpace := maxWidth - totalFieldLen
	numGaps := len(fields) - 1

	if numGaps == 0 {
		return fields[0]
	}

	// Distribute space equally between fields
	spacePerGap := availableSpace / numGaps
	if spacePerGap < 1 {
		spacePerGap = 1 // minimum 1 space between fields
	}

	// Build the line with distributed spacing
	var result strings.Builder
	for i, field := range fields {
		result.WriteString(field)
		if i < len(fields)-1 {
			result.WriteString(strings.Repeat(" ", spacePerGap))
		}
	}

	return result.String()
}

func (l *zapLogger) formatLog(level string, msg string, fields []logger.Field) string {
	pc, file, line, _ := runtime.Caller(callerSkipFormatLog)
	fn := runtime.FuncForPC(pc)
	funcName := "unknown"
	if fn != nil {
		funcName = fn.Name()
		if lastSlash := strings.LastIndex(funcName, "/"); lastSlash != -1 {
			funcName = funcName[lastSlash+1:]
		}
	}

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

	dims := calculateStandardBoxDimensions(l.termWidth)

	// Combine message with formatted fields
	message := msg
	fieldLines := formatFields(fields, dims.Message.Width)
	if len(fieldLines) > 0 {
		message = msg + "\n" + strings.Join(fieldLines, "\n")
	}

	return l.drawBoxedLog(timestamp, level, fileLoc, service, message)
}

func (l *zapLogger) Debug(msg string, fields ...logger.Field) {
	if l.consoleMode && l.logger.Core().Enabled(zapcore.DebugLevel) {
		fmt.Print(l.formatLog("DEBUG", msg, fields))
		return
	}
	l.logger.Debug(msg, l.makeZapFields(fields)...)
}

func (l *zapLogger) Info(msg string, fields ...logger.Field) {
	if l.consoleMode && l.logger.Core().Enabled(zapcore.InfoLevel) {
		fmt.Print(l.formatLog("INFO", msg, fields))
		return
	}
	l.logger.Info(msg, l.makeZapFields(fields)...)
}

func (l *zapLogger) Warn(msg string, fields ...logger.Field) {
	if l.consoleMode && l.logger.Core().Enabled(zapcore.WarnLevel) {
		fmt.Print(l.formatLog("WARN", msg, fields))
		return
	}
	l.logger.Warn(msg, l.makeZapFields(fields)...)
}

func (l *zapLogger) Error(msg string, fields ...logger.Field) {
	if l.consoleMode && l.logger.Core().Enabled(zapcore.ErrorLevel) {
		// Don't pass fields to formatLog since they're already handled there
		content := msg + "\n" + customStackTrace(callerSkipError, 0)
		fmt.Print(l.formatLog("ERROR", content, fields))
		return
	}
	zfs := l.makeZapFields(fields)

	zfs = append(zfs, zap.String("stack", customStackTrace(callerSkipError, 0)))
	l.logger.Error(msg, zfs...)
}

func (l *zapLogger) Fatal(msg string, fields ...logger.Field) {
	if l.consoleMode {
		// Don't pass fields to formatLog since they're already handled there
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
	if l.consoleMode {
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

		dims := calculateRequestBoxDimensions(l.termWidth)

		message := msg
		if len(fields) > 0 {
			extraFields := []logger.Field{}
			for _, f := range fields {
				if f != nil && f.Key() != "status" {
					extraFields = append(extraFields, f)
				}
			}
			if len(extraFields) > 0 {
				fieldLines := formatFields(extraFields, dims.Message.Width)
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
		zap.Int64("duration_ms", duration.Milliseconds()),
		zap.String("service", l.service),
	)
	l.logger.Info(msg, zfs...)
}

func (l *zapLogger) With(fields ...logger.Field) logger.Logger {
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
