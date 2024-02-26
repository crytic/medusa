package cmd

import (
	"fmt"
	"github.com/crytic/medusa/cmd/exitcodes"
	"github.com/crytic/medusa/logging/colors"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/crytic/medusa/fuzzing"
	"github.com/crytic/medusa/fuzzing/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// fuzzCmd represents the command provider for fuzzing
var fuzzCmd = &cobra.Command{
	Use:               "fuzz",
	Short:             "Starts a fuzzing campaign",
	Long:              `Starts a fuzzing campaign`,
	Args:              cmdValidateFuzzArgs,
	ValidArgsFunction: cmdValidFuzzArgs,
	RunE:              cmdRunFuzz,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

func init() {
	// Add all the flags allowed for the fuzz command
	err := addFuzzFlags()
	if err != nil {
		cmdLogger.Panic("Failed to initialize the fuzz command", err)
	}

	// Add the fuzz command and its associated flags to the root command
	rootCmd.AddCommand(fuzzCmd)
}

// cmdValidFuzzArgs will return which flags and sub-commands are valid for dynamic completion for the fuzz command
func cmdValidFuzzArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Gather a list of flags that are available to be used in the current command but have not been used yet
	var unusedFlags []string

	// Examine all the flags, and add any flags that have not been set in the current command line
	// to a list of unused flags
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if !flag.Changed {
			// When adding a flag to a command, include the "--" prefix to indicate that it is a flag
			// and not a positional argument. Additionally, when the user presses the TAB key twice after typing
			// a flag name, the "--" prefix will appear again, indicating that more flags are available and that
			// none of the arguments are positional.
			unusedFlags = append(unusedFlags, "--"+flag.Name)
		}
	})
	// Provide a list of flags that can be used in the current command (but have not been used yet)
	// for autocompletion suggestions
	return unusedFlags, cobra.ShellCompDirectiveNoFileComp
}

// cmdValidateFuzzArgs makes sure that there are no positional arguments provided to the fuzz command
func cmdValidateFuzzArgs(cmd *cobra.Command, args []string) error {
	// Make sure we have no positional args
	if err := cobra.NoArgs(cmd, args); err != nil {
		err = fmt.Errorf("fuzz does not accept any positional arguments, only flags and their associated values")
		cmdLogger.Error("Failed to validate args to the fuzz command", err)
		return err
	}
	return nil
}

// cmdRunFuzz executes the CLI fuzz command and navigates through the following possibilities:
// #1: We will search for either a custom config file (via --config) or the default (medusa.json).
// If we find it, read it. If we can't read it, throw an error.
// #2: If a custom file was provided (--config was used), and we can't find the file, throw an error.
// #3: If medusa.json can't be found, use the default project configuration.
func cmdRunFuzz(cmd *cobra.Command, args []string) error {
	var projectConfig *config.ProjectConfig

	// Check to see if --config flag was used and store the value of --config flag
	configFlagUsed := cmd.Flags().Changed("config")
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		cmdLogger.Error("Failed to run the fuzz command", err)
		return err
	}

	// If --config was not used, look for `medusa.json` in the current work directory
	if !configFlagUsed {
		workingDirectory, err := os.Getwd()
		if err != nil {
			cmdLogger.Error("Failed to run the fuzz command", err)
			return err
		}
		configPath = filepath.Join(workingDirectory, DefaultProjectConfigFilename)
	}

	// Check to see if the file exists at configPath
	_, existenceError := os.Stat(configPath)

	// Possibility #1: File was found
	if existenceError == nil {
		// Try to read the configuration file and throw an error if something goes wrong
		cmdLogger.Info("Reading the configuration file at: ", colors.Bold, configPath, colors.Reset)
		projectConfig, err = config.ReadProjectConfigFromFile(configPath)
		if err != nil {
			cmdLogger.Error("Failed to run the fuzz command", err)
			return err
		}
	}

	// Possibility #2: If the --config flag was used, and we couldn't find the file, we'll throw an error
	if configFlagUsed && existenceError != nil {
		cmdLogger.Error("Failed to run the fuzz command", err)
		return existenceError
	}

	// Possibility #3: --config flag was not used and medusa.json was not found, so use the default project config
	if !configFlagUsed && existenceError != nil {
		cmdLogger.Warn(fmt.Sprintf("Unable to find the config file at %v, will use the default project configuration for the "+
			"%v compilation platform instead", configPath, DefaultCompilationPlatform))

		projectConfig, err = config.GetDefaultProjectConfig(DefaultCompilationPlatform)
		if err != nil {
			cmdLogger.Error("Failed to run the fuzz command", err)
			return err
		}
	}

	// Update the project configuration given whatever flags were set using the CLI
	err = updateProjectConfigWithFuzzFlags(cmd, projectConfig)
	if err != nil {
		cmdLogger.Error("Failed to run the fuzz command", err)
		return err
	}

	// Change our working directory to the parent directory of the project configuration file
	// This is important as when we compile for a given platform, the paths may be relative to wherever the
	// configuration is supplied from. Providing a file path explicitly is optional anyways, so we _should_
	// be in the config directory when running this.
	err = os.Chdir(filepath.Dir(configPath))
	if err != nil {
		cmdLogger.Error("Failed to run the fuzz command", err)
		return err
	}

	// Create our fuzzing
	fuzzer, err := fuzzing.NewFuzzer(*projectConfig)
	if err != nil {
		return err
	}

	// Stop our fuzzing on keyboard interrupts
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		fuzzer.Stop()
	}()

	// Start the fuzzing process with our cancellable context.
	err = fuzzer.Start()

	// If we have no error and failed test cases, we'll want to return a special exit code
	if err == nil && len(fuzzer.TestCasesWithStatus(fuzzing.TestCaseStatusFailed)) > 0 {
		return exitcodes.NewErrorWithExitCode(err, exitcodes.ExitCodeTestFailed)
	}

	return err
}
