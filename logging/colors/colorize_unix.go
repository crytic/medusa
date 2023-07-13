//go:build !windows
// +build !windows

package colors

import "fmt"

// EnableColor is a no-op function for non-windows systems because we know that they support ANSI escape codes
func EnableColor() {}

// Colorize returns the string s wrapped in ANSI code c for non-windows systems
// Source: https://github.com/rs/zerolog/blob/4fff5db29c3403bc26dee9895e12a108aacc0203/console.go
func Colorize(s any, c Color) string {
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}
