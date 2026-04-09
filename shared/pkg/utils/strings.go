package utils

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
