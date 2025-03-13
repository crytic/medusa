package utils

// MapFetchCasted obtains a key from a given map, automatically casting its value.
// Returns the value as the correct type, or nil if it could not be found or type converted.
func MapFetchCasted[K comparable, V any](m map[K]any, key K) *V {
	// Try to obtain the result
	if genericResult, ok := m[key]; ok {
		if castedResult, ok := genericResult.(V); ok {
			return &castedResult
		}
	}
	return nil
}
