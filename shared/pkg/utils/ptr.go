package utils

func DerefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func DerefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func DerefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func DerefFloat64(f *float64) float64 {
	if f == nil {
		return 0.0
	}
	return *f
}

func DerefInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func PtrString(s string) *string {
	return &s
}

func PtrInt(i int) *int {
	return &i
}

func PtrBool(b bool) *bool {
	return &b
}

func PtrFloat64(f float64) *float64 {
	return &f
}

func PtrInt64(i int64) *int64 {
	return &i
}
