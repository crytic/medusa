package utils

// GetAndRemoveKeyFromMapping will retrieve the value for a given key in the mapping and will also delete the key-value
// pair from the return. If the key does not exist, a nil value is returned with the unmodified mapping.
func GetAndRemoveKeyFromMapping(mapping map[string]any, key string) (any, map[string]any) {
	// Guard clause to make sure we don't access a nil-mapping
	if mapping == nil {
		return nil, nil
	}

	// Grab key and return nil if the key does not exist
	val, found := mapping[key]
	if !found {
		return nil, mapping
	}

	// Remove the key from the fields
	delete(mapping, key)

	return val, mapping
}
