//go:build windows
// +build windows

package colors

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

var enabled bool

// EnableColor will make a kernel call to enable ANSI escape codes on both stdout and stderr. Note that if enablement
// on either stream fails, then coloring will not be enabled.
func EnableColor() {
	var mode uint32
	fds := []uintptr{os.Stdout.Fd(), os.Stderr.Fd()}

	// Iterate across each file descriptor and enable coloring
	for _, fd := range fds {
		// Obtain our current console mode.
		consoleHandle := windows.Handle(fd)
		err := windows.GetConsoleMode(consoleHandle, &mode)
		if err != nil {
			enabled = false
			return
		}

		// If color is not enabled, try to enable it.
		if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING == 0 {
			err = windows.SetConsoleMode(consoleHandle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
			if err != nil {
				enabled = false
				return
			}
		}

		// Fetch the console mode once more
		err = windows.GetConsoleMode(consoleHandle, &mode)
		if err != nil {
			enabled = false
			return
		}

		// Set our enabled status after trying to enable it.
		enabled = mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0

		// If we failed to enable on this file descriptor, exit out
		if !enabled {
			return
		}
	}
}

// DisableColor will disable colors
func DisableColor() { enabled = false }

// Colorize returns the string s wrapped in ANSI code c assuming that ANSI is supported on the Windows version
// Source: https://github.com/rs/zerolog/blob/4fff5db29c3403bc26dee9895e12a108aacc0203/console.go
func Colorize(s any, c Color) string {
	// If ANSI is not supported then just return the original string
	if !enabled {
		return fmt.Sprintf("%v", s)
	}

	// Otherwise, returned an ANSI-wrapped string
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}
