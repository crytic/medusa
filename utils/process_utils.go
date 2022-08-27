package utils

import (
	"bytes"
	"os/exec"
)

// RunCommandWithOutputAndError runs a given exec.Cmd and returns the stdout and stderr output
// as bytes, or an error if one occurred.
func RunCommandWithOutputAndError(command *exec.Cmd) ([]byte, []byte, error) {
	// Create our buffers to capture output and errors.
	var bStdout, bStderr bytes.Buffer
	command.Stdout = &bStdout
	command.Stderr = &bStderr

	// Execute the command and perform error checking
	err := command.Run()

	// Set our results
	stdout := bStdout.Bytes()
	stderr := bStderr.Bytes()
	return stdout, stderr, err
}
