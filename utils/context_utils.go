package utils

import "golang.org/x/net/context"

// CheckContextDone checks if a provided context has indicated it is done, and returns a boolean indicating if it is.
func CheckContextDone(ctx context.Context) bool {
	// Check if the context is done in a non-blocking fashion.
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
