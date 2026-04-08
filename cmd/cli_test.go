package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crytic/medusa/fuzzing/config"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFuzzCommandForTest returns a fresh fuzz command with its flags registered.
func newFuzzCommandForTest(t *testing.T) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{Use: "fuzz"}
	require.NoError(t, addFuzzFlagsToCommand(cmd))
	return cmd
}

// newInitCommandForTest returns a fresh init command with its flags registered.
func newInitCommandForTest(t *testing.T) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{Use: "init"}
	require.NoError(t, addInitFlagsToCommand(cmd))
	return cmd
}

// newCorpusCleanCommandForTest returns a fresh corpus clean command with its flags registered.
func newCorpusCleanCommandForTest(t *testing.T) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{Use: "clean"}
	require.NoError(t, addCorpusCleanFlagsToCommand(cmd))
	return cmd
}

// changeWorkingDirectory changes the working directory for a test and restores it afterwards.
func changeWorkingDirectory(t *testing.T, dir string) {
	t.Helper()

	currentDirectory, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))

	t.Cleanup(func() {
		require.NoError(t, os.Chdir(currentDirectory))
	})
}

// setTestStdin replaces stdin for the duration of a test.
func setTestStdin(t *testing.T, input string) {
	t.Helper()

	originalStdin := os.Stdin
	stdinFile, err := os.CreateTemp(t.TempDir(), "stdin-*")
	require.NoError(t, err)

	_, err = stdinFile.WriteString(input)
	require.NoError(t, err)
	_, err = stdinFile.Seek(0, 0)
	require.NoError(t, err)

	os.Stdin = stdinFile
	t.Cleanup(func() {
		os.Stdin = originalStdin
		require.NoError(t, stdinFile.Close())
	})
}

// writeProjectConfig writes a project config to disk and returns the output path.
func writeProjectConfig(t *testing.T, dir string, update func(*config.ProjectConfig)) string {
	t.Helper()

	projectConfig, err := config.GetDefaultProjectConfig(DefaultCompilationPlatform)
	require.NoError(t, err)

	if update != nil {
		update(projectConfig)
	}

	configPath := filepath.Join(dir, DefaultProjectConfigFilename)
	require.NoError(t, projectConfig.WriteToFile(configPath))
	return configPath
}

func TestCmdValidateFuzzArgsRejectsPositionalArgs(t *testing.T) {
	cmd := newFuzzCommandForTest(t)

	err := cmdValidateFuzzArgs(cmd, []string{"unexpected"})

	require.Error(t, err)
	assert.EqualError(t, err, "fuzz does not accept any positional arguments, only flags and their associated values")
}

func TestCmdValidFuzzArgsReturnsOnlyUnusedFlags(t *testing.T) {
	cmd := newFuzzCommandForTest(t)
	require.NoError(t, cmd.ParseFlags([]string{"--workers", "2", "--fail-fast"}))

	args, directive := cmdValidFuzzArgs(cmd, nil, "")

	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.NotContains(t, args, "--workers")
	assert.NotContains(t, args, "--fail-fast")
	assert.Contains(t, args, "--config")
	assert.Contains(t, args, "--log-level")
}

func TestUpdateProjectConfigWithFuzzFlagsAppliesCLIOverrides(t *testing.T) {
	projectConfig, err := config.GetDefaultProjectConfig(DefaultCompilationPlatform)
	require.NoError(t, err)

	cmd := newFuzzCommandForTest(t)
	require.NoError(t, cmd.ParseFlags([]string{
		"--compilation-target", "contracts",
		"--workers", "21",
		"--timeout", "33",
		"--test-limit", "44",
		"--seq-len", "55",
		"--target-contracts", "Alpha,Beta",
		"--corpus-dir", "corpus",
		"--senders", "0x10001,0x10002",
		"--deployer", "0x10003",
		"--no-color",
		"--fail-fast",
		"--explore",
		"--use-slither-force",
		"--rpc-url", "http://127.0.0.1:8545",
		"--rpc-block", "1234",
		"-vvv",
		"--log-level", "debug",
	}))

	require.NoError(t, updateProjectConfigWithFuzzFlags(cmd, projectConfig))

	platformConfig, err := projectConfig.Compilation.GetPlatformConfig()
	require.NoError(t, err)

	assert.Equal(t, "contracts", platformConfig.GetTarget())
	assert.Equal(t, 21, projectConfig.Fuzzing.Workers)
	assert.Equal(t, 33, projectConfig.Fuzzing.Timeout)
	assert.EqualValues(t, 44, projectConfig.Fuzzing.TestLimit)
	assert.Equal(t, 55, projectConfig.Fuzzing.CallSequenceLength)
	assert.Equal(t, []string{"Alpha", "Beta"}, projectConfig.Fuzzing.TargetContracts)
	assert.Equal(t, "corpus", projectConfig.Fuzzing.CorpusDirectory)
	assert.Equal(t, []string{"0x10001", "0x10002"}, projectConfig.Fuzzing.SenderAddresses)
	assert.Equal(t, "0x10003", projectConfig.Fuzzing.DeployerAddress)
	assert.True(t, projectConfig.Logging.NoColor)
	assert.False(t, projectConfig.Fuzzing.Testing.StopOnFailedTest)
	assert.False(t, projectConfig.Fuzzing.Testing.StopOnNoTests)
	assert.False(t, projectConfig.Fuzzing.Testing.AssertionTesting.Enabled)
	assert.False(t, projectConfig.Fuzzing.Testing.PropertyTesting.Enabled)
	assert.False(t, projectConfig.Fuzzing.Testing.OptimizationTesting.Enabled)
	assert.True(t, projectConfig.Slither.UseSlither)
	assert.True(t, projectConfig.Slither.OverwriteCache)
	assert.True(t, projectConfig.Fuzzing.TestChainConfig.ForkConfig.ForkModeEnabled)
	assert.Equal(t, "http://127.0.0.1:8545", projectConfig.Fuzzing.TestChainConfig.ForkConfig.RpcUrl)
	assert.EqualValues(t, 1234, projectConfig.Fuzzing.TestChainConfig.ForkConfig.RpcBlock)
	assert.Equal(t, config.VeryVeryVerbose, projectConfig.Fuzzing.Testing.Verbosity)
	assert.Equal(t, zerolog.DebugLevel, projectConfig.Logging.Level)
}

func TestUpdateProjectConfigWithFuzzFlagsRejectsInvalidLogLevel(t *testing.T) {
	projectConfig, err := config.GetDefaultProjectConfig(DefaultCompilationPlatform)
	require.NoError(t, err)

	cmd := newFuzzCommandForTest(t)
	require.NoError(t, cmd.ParseFlags([]string{"--log-level", "fatal"}))

	err = updateProjectConfigWithFuzzFlags(cmd, projectConfig)

	require.Error(t, err)
	assert.EqualError(t, err, "invalid log level (expected trace, debug, info, warn, error, or panic)")
}

func TestCmdRunFuzzErrorsWhenExplicitConfigIsMissing(t *testing.T) {
	cmd := newFuzzCommandForTest(t)
	missingConfigPath := filepath.Join(t.TempDir(), "missing.json")
	require.NoError(t, cmd.ParseFlags([]string{"--config", missingConfigPath}))

	err := cmdRunFuzz(cmd, nil)

	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestCmdRunFuzzUsesDefaultConfigBeforeFuzzerCreation(t *testing.T) {
	tempDir := t.TempDir()
	changeWorkingDirectory(t, tempDir)

	cmd := newFuzzCommandForTest(t)
	require.NoError(t, cmd.ParseFlags([]string{"--log-level", "fatal"}))

	err := cmdRunFuzz(cmd, nil)

	require.Error(t, err)
	assert.EqualError(t, err, "invalid log level (expected trace, debug, info, warn, error, or panic)")
}

func TestCmdValidateInitArgsRejectsUnsupportedPlatforms(t *testing.T) {
	cmd := newInitCommandForTest(t)

	err := cmdValidateInitArgs(cmd, []string{"unsupported"})

	require.Error(t, err)
	assert.EqualError(t, err, "init was provided invalid platform argument 'unsupported' (options: "+strings.Join(supportedPlatforms, ", ")+")")
}

func TestCmdValidateInitArgsRejectsTooManyArgs(t *testing.T) {
	cmd := newInitCommandForTest(t)

	err := cmdValidateInitArgs(cmd, []string{"solc", "extra"})

	require.Error(t, err)
	assert.EqualError(t, err, "init accepts at most 1 platform argument (options: "+strings.Join(supportedPlatforms, ", ")+"). default platform is "+DefaultCompilationPlatform+"\n")
}

func TestCmdValidInitArgsIncludesPlatformsUntilAFlagIsUsed(t *testing.T) {
	cmd := newInitCommandForTest(t)

	args, directive := cmdValidInitArgs(cmd, nil, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.Contains(t, args, "--out")
	assert.Contains(t, args, "crytic-compile")
	assert.Contains(t, args, "solc")

	require.NoError(t, cmd.ParseFlags([]string{"--out", "medusa.json"}))

	args, directive = cmdValidInitArgs(cmd, nil, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	assert.Contains(t, args, "--compilation-target")
	assert.NotContains(t, args, "--out")
	assert.NotContains(t, args, "crytic-compile")
	assert.NotContains(t, args, "solc")
}

func TestCmdRunInitWritesConfigForSelectedPlatform(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "medusa.json")
	cmd := newInitCommandForTest(t)
	require.NoError(t, cmd.ParseFlags([]string{
		"--out", outputPath,
		"--compilation-target", "contracts",
	}))

	require.NoError(t, cmdRunInit(cmd, []string{"solc"}))

	projectConfig, err := config.ReadProjectConfigFromFile(outputPath, "solc")
	require.NoError(t, err)

	platformConfig, err := projectConfig.Compilation.GetPlatformConfig()
	require.NoError(t, err)

	assert.Equal(t, "solc", projectConfig.Compilation.Platform)
	assert.Equal(t, "contracts", platformConfig.GetTarget())
}

func TestCmdRunInitSkipsOverwriteWhenUserDeclines(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "medusa.json")
	require.NoError(t, os.WriteFile(outputPath, []byte("existing config"), 0o644))

	setTestStdin(t, "n\n")

	cmd := newInitCommandForTest(t)
	require.NoError(t, cmd.ParseFlags([]string{"--out", outputPath}))

	require.NoError(t, cmdRunInit(cmd, nil))

	fileContents, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, "existing config", string(fileContents))
}

func TestCmdRunCorpusCleanErrorsWhenCorpusDirectoryIsNotConfigured(t *testing.T) {
	tempDir := t.TempDir()
	configPath := writeProjectConfig(t, tempDir, nil)
	changeWorkingDirectory(t, tempDir)
	expectedConfigPath, err := filepath.EvalSymlinks(configPath)
	require.NoError(t, err)

	cmd := newCorpusCleanCommandForTest(t)

	err = cmdRunCorpusClean(cmd, nil)

	require.Error(t, err)
	assert.EqualError(t, err, "no corpus directory configured in "+expectedConfigPath)
}

func TestCmdRunCorpusCleanErrorsWhenConfiguredCorpusDirectoryDoesNotExist(t *testing.T) {
	tempDir := t.TempDir()
	configPath := writeProjectConfig(t, tempDir, func(projectConfig *config.ProjectConfig) {
		projectConfig.Fuzzing.CorpusDirectory = "missing-corpus"
	})
	changeWorkingDirectory(t, tempDir)

	cmd := newCorpusCleanCommandForTest(t)
	require.NoError(t, cmd.ParseFlags([]string{"--config", configPath}))

	err := cmdRunCorpusClean(cmd, nil)

	require.Error(t, err)
	assert.EqualError(t, err, "corpus directory does not exist: missing-corpus")
}
