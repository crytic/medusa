//go:build windows
// +build windows

package colors

import (
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
)

var enabled bool

// EnableColor will make a kernel call to see whether ANSI escape codes are supported on the stdout channel for the
// Windows system.
func EnableColor() {
	var mode uint32
	// If mode does not equal windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING then the stdout does not support ANSI escape codes
	if r, _, _ := procGetConsoleMode.Call(os.Stdout.Fd(), uintptr(unsafe.Pointer(&mode))); r != 0 && mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		enabled = false
	} else {
		enabled = true
	}
}

// Colorize returns the string s wrapped in ANSI code c assuming that ANSI is supported on the Windows version
// Source: https://github.com/rs/zerolog/blob/4fff5db29c3403bc26dee9895e12a108aacc0203/console.go
func Colorize(s any, c Color) string {
	// If ANSI is not supported then just return the original string
	if !enabled {
		return fmt.Sprintf("%s", s)
	}

	// Otherwise, returned an ANSI-wrapped string
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}
