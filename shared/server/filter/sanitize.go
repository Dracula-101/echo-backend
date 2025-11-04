package filter

import (
	"html"
	"regexp"
	"strings"
)

var (
	// SQL injection patterns
	sqlPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute|script|javascript|<script)`),
		regexp.MustCompile(`(?i)(--|;|\/\*|\*\/|xp_|sp_)`),
	}

	// XSS patterns
	xssPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)on\w+\s*=`), // onclick, onload, etc.
	}

	// Path traversal patterns
	pathTraversalPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\.\.(/|\\)`),
		regexp.MustCompile(`%2e%2e`),
	}
)

// SanitizeString removes potentially dangerous characters from a string
func SanitizeString(input string) string {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// HTML escape
	input = html.EscapeString(input)

	return input
}

// SanitizeHTML removes dangerous HTML/JavaScript from input
func SanitizeHTML(input string) string {
	result := input

	// Remove script tags and content
	for _, pattern := range xssPatterns {
		result = pattern.ReplaceAllString(result, "")
	}

	// HTML escape the result
	result = html.EscapeString(result)

	return result
}

// ContainsSQLInjection checks if input contains SQL injection patterns
func ContainsSQLInjection(input string) bool {
	for _, pattern := range sqlPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// ContainsXSS checks if input contains XSS patterns
func ContainsXSS(input string) bool {
	for _, pattern := range xssPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// ContainsPathTraversal checks if input contains path traversal patterns
func ContainsPathTraversal(input string) bool {
	for _, pattern := range pathTraversalPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// IsDangerous checks if input contains any dangerous patterns
func IsDangerous(input string) bool {
	return ContainsSQLInjection(input) || ContainsXSS(input) || ContainsPathTraversal(input)
}

// CleanFilename sanitizes a filename
func CleanFilename(filename string) string {
	// Remove path traversal attempts
	filename = strings.ReplaceAll(filename, "..", "")
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")

	// Remove null bytes and control characters
	filename = strings.Map(func(r rune) rune {
		if r == 0 || r < 32 {
			return -1
		}
		return r
	}, filename)

	// Limit filename length
	if len(filename) > 255 {
		filename = filename[:255]
	}

	return strings.TrimSpace(filename)
}

// TruncateString truncates a string to a maximum length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// RemoveInvisibleChars removes invisible characters from string
func RemoveInvisibleChars(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, s)
}

// NormalizeWhitespace normalizes whitespace in a string
func NormalizeWhitespace(s string) string {
	// Replace multiple spaces with single space
	re := regexp.MustCompile(`\s+`)
	s = re.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
