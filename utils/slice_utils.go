package utils

// SlicePointersToValues takes a slice of pointers and returns a slice of values de-referenced from them.
func SlicePointersToValues[T any](x []*T) []T {
	r := make([]T, len(x))
	for i := 0; i < len(x); i++ {
		r[i] = *x[i]
	}
	return r
}

// SliceValuesToPointers takes a slice of values and returns a slice of pointers to them.
func SliceValuesToPointers[T any](x []T) []*T {
	r := make([]*T, len(x))
	for i := 0; i < len(x); i++ {
		r[i] = &x[i]
	}
	return r
}

// SliceSelect provides a way of querying a specific element from a slice's elements into a slice of its own.
func SliceSelect[T any, K any](x []T, f func(x T) K) []K {
	r := make([]K, len(x))
	for i := 0; i < len(x); i++ {
		r[i] = f(x[i])
	}
	return r
}

// SliceWhere provides a way of querying specific elements which fit some criteria into a new slice.
func SliceWhere[T any](x []T, f func(x T) bool) []T {
	r := make([]T, 0)
	for i := 0; i < len(x); i++ {
		if f(x[i]) {
			r = append(r, x[i])
		}
	}
	return r
}
