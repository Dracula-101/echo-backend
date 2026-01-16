package utils

// get method for map with default value
func GetValue[K comparable, V any](m map[K]V, key K, defaultValue V) V {
	if val, exists := m[key]; exists {
		return val
	}
	return defaultValue
}
