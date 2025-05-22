package utils

import (
	"bytes"
	"io"
	"os/exec"
	"runtime"
	"sync"
)

// RunCommandWithOutputAndError runs a given exec.Cmd and returns the stdout, stderr, and
// combined output as bytes, or an error if one occurred.
func RunCommandWithOutputAndError(command *exec.Cmd) ([]byte, []byte, []byte, error) {
	// Create our buffers to capture output and errors.
	var bStdout, bStderr, bCombined bytes.Buffer

	// Create a synchronized writer over bCombined to avoid data race.
	var combinedWriter io.Writer = &synchronizedWriter{writer: &bCombined}

	// Create multi writers to capture output into individual and combined buffers
	stdoutMulti := io.MultiWriter(&bStdout, combinedWriter)
	stderrMulti := io.MultiWriter(&bStderr, combinedWriter)

	// Set our writers
	command.Stdout = stdoutMulti
	command.Stderr = stderrMulti

	// Execute the command
	err := command.Run()

	// Return our results
	return bStdout.Bytes(), bStderr.Bytes(), bCombined.Bytes(), err
}

// IsWindowsEnvironment returns a boolean indicating whether the current execution environment is a Windows platform.
func IsWindowsEnvironment() bool {
	return runtime.GOOS == "windows"
}

// IsMacOSEnvironment returns a boolean indicating whether the current execution environment is a macOS platform.
func IsMacOSEnvironment() bool {
	return runtime.GOOS == "darwin"
}

// IsLinuxEnvironment returns a boolean indicating whether the current execution environment is a Linux platform.
func IsLinuxEnvironment() bool {
	return runtime.GOOS == "linux"
}

// synchronizedWriter wraps an io.Writer to avoid a data race when writing.
type synchronizedWriter struct {
	writer io.Writer
	mutex  sync.Mutex
}

func (s *synchronizedWriter) Write(p []byte) (n int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.writer.Write(p)
}
