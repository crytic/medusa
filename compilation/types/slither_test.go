package types

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunSlitherCommandIgnoresStderr verifies diagnostics do not contaminate Slither's JSON output.
func TestRunSlitherCommandIgnoresStderr(t *testing.T) {
	stdout := `{"success":true,"error":"","results":{"printers":[{"description":"{\"constants_used\":{}}"}]}}`
	cmd := newSlitherHelperCommand(t, stdout, "compiler warning\n", 0)

	out, err := runSlitherCommand(cmd)
	require.NoError(t, err)
	assert.Equal(t, stdout, string(out))

	var results SlitherResults
	require.NoError(t, results.UnmarshalJSON(out))
	assert.Empty(t, results.Constants)
}

// TestRunSlitherCommandReportsStderr verifies command diagnostics remain available when Slither fails.
func TestRunSlitherCommandReportsStderr(t *testing.T) {
	cmd := newSlitherHelperCommand(t, "", "slither failed to compile\n", 1)

	_, err := runSlitherCommand(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "slither failed to compile")
}

// newSlitherHelperCommand creates a cross-platform subprocess that emits controlled output for command tests.
func newSlitherHelperCommand(t *testing.T, stdout string, stderr string, exitCode int) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestSlitherHelperProcess$")
	cmd.Env = append(os.Environ(),
		"GO_WANT_SLITHER_HELPER_PROCESS=1",
		"SLITHER_HELPER_STDOUT="+stdout,
		"SLITHER_HELPER_STDERR="+stderr,
		"SLITHER_HELPER_EXIT_CODE="+strconv.Itoa(exitCode),
	)
	return cmd
}

// TestSlitherHelperProcess emits subprocess output configured by newSlitherHelperCommand.
func TestSlitherHelperProcess(t *testing.T) {
	// Return immediately when this test is not running as a helper subprocess.
	if os.Getenv("GO_WANT_SLITHER_HELPER_PROCESS") != "1" {
		return
	}

	// Write the configured values to their respective process streams.
	_, _ = fmt.Fprint(os.Stdout, os.Getenv("SLITHER_HELPER_STDOUT"))
	_, _ = fmt.Fprint(os.Stderr, os.Getenv("SLITHER_HELPER_STDERR"))

	// Exit with the configured status so callers can exercise success and failure paths.
	exitCode, err := strconv.Atoi(os.Getenv("SLITHER_HELPER_EXIT_CODE"))
	if err != nil {
		_, _ = fmt.Fprint(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(exitCode)
}
