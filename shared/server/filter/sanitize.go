package filter

import (
	"html"
	"regexp"
	"strings"
)

var (
	sqlPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute|script|javascript|<script)`),
		regexp.MustCompile(`(?i)(--|;|\/\*|\*\/|xp_|sp_)`),
	}

	xssPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
		regexp.MustCompile(`(?i)javascript:`),
		regexp.MustCompile(`(?i)on\w+\s*=`), // onclick, onload, etc.
	}

	pathTraversalPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\.\.(/|\\)`),
		regexp.MustCompile(`%2e%2e`),
	}
)

func SanitizeString(input string) string {
	input = strings.TrimSpace(input)
	input = strings.ReplaceAll(input, "\x00", "")
	input = html.EscapeString(input)
	return input
}

func SanitizeHTML(input string) string {
	result := input

	for _, pattern := range xssPatterns {
		result = pattern.ReplaceAllString(result, "")
	}
	result = html.EscapeString(result)
	return result
}

func ContainsSQLInjection(input string) bool {
	for _, pattern := range sqlPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

func ContainsXSS(input string) bool {
	for _, pattern := range xssPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

func ContainsPathTraversal(input string) bool {
	for _, pattern := range pathTraversalPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

func IsDangerous(input string) bool {
	return ContainsSQLInjection(input) || ContainsXSS(input) || ContainsPathTraversal(input)
}

func CleanFilename(filename string) string {
	filename = strings.ReplaceAll(filename, "..", "")
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")

	filename = strings.Map(func(r rune) rune {
		if r == 0 || r < 32 {
			return -1
		}
		return r
	}, filename)

	if len(filename) > 255 {
		filename = filename[:255]
	}

	return strings.TrimSpace(filename)
}

func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func RemoveInvisibleChars(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, s)
}

func NormalizeWhitespace(s string) string {
	re := regexp.MustCompile(`\s+`)
	s = re.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
