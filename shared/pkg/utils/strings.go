package utils

import "html"

func SafeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func StringToBytes(s string) []byte {
	return []byte(s)
}

func BytesToString(b []byte) string {
	return string(b)
}

func SanitizeHTMLText(s string) string {
	return html.EscapeString(s)
}

func TrimString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength]
}
