package types

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// newCachedSlitherConfig writes cached Slither output to disk and returns a config that reads it back.
func newCachedSlitherConfig(t *testing.T, contents string) *SlitherConfig {
	t.Helper()
	cachePath := filepath.Join(t.TempDir(), "slither_results.json")
	require.NoError(t, os.WriteFile(cachePath, []byte(contents), 0644))
	return &SlitherConfig{UseSlither: true, CachePath: cachePath, Args: []string{}}
}

// TestRunSlitherEmptyPrintersReturnsError verifies cached output carrying no printer results is reported as an
// error so the fuzzer can fall back to seeding from the AST.
func TestRunSlitherEmptyPrintersReturnsError(t *testing.T) {
	config := newCachedSlitherConfig(t, `{"success":true,"error":"","results":{"printers":[]}}`)

	var err error
	require.NotPanics(t, func() {
		_, err = config.RunSlither("target", "")
	})
	assert.Error(t, err)
}

// TestRunSlitherEmptyConstantIsSkipped verifies a constant entry with no values is skipped while the remaining
// constants are still collected.
func TestRunSlitherEmptyConstantIsSkipped(t *testing.T) {
	description := `{\"constants_used\":{\"C\":{\"f()\":[[],[{\"type\":\"uint256\",\"value\":\"7\"}]]}}}`
	config := newCachedSlitherConfig(t, `{"success":true,"error":"","results":{"printers":[{"description":"`+description+`"}]}}`)

	var (
		results *SlitherResults
		err     error
	)
	require.NotPanics(t, func() {
		results, err = config.RunSlither("target", "")
	})
	require.NoError(t, err)
	require.NotNil(t, results)
	assert.Equal(t, []Constant{{Type: "uint256", Value: "7"}}, results.Constants)
}
