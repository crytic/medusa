package utils

import (
	"bytes"
	"io"
	"os/exec"
)

// RunCommandWithOutputAndError runs a given exec.Cmd and returns the stdout, stderr, and
// combined output as bytes, or an error if one occurred.
func RunCommandWithOutputAndError(command *exec.Cmd) ([]byte, []byte, []byte, error) {
	// Create our buffers to capture output and errors.
	var bStdout, bStderr, bCombined bytes.Buffer

	// Create multi writers to capture output into individual and combined buffers
	stdoutMulti := io.MultiWriter(&bStdout, &bCombined)
	stderrMulti := io.MultiWriter(&bStderr, &bCombined)

	// Set our writers
	command.Stdout = stdoutMulti
	command.Stderr = stderrMulti

	// Execute the command
	err := command.Run()

	// Return our results
	return bStdout.Bytes(), bStderr.Bytes(), bCombined.Bytes(), err
}
