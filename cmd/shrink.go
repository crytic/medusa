package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/crytic/medusa/cmd/exitcodes"
	"github.com/crytic/medusa/logging/colors"

	"github.com/crytic/medusa/fuzzing"
	"github.com/crytic/medusa/fuzzing/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// shrinkCmd represents the command provider for fuzzing
var shrinkCmd = &cobra.Command{
	Use:               "shrink",
	Short:             "Shrink transaction(s)",
	Long:              `Shrink transaction(s)`,
	Args:              cmdValidateShrinkArgs,
	ValidArgsFunction: cmdValidShrinkArgs,
	RunE:              cmdRunShrink,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

func init() {
	// Add all the flags allowed for the shrink command
	err := addShrinkFlags()
	if err != nil {
		cmdLogger.Panic("Failed to initialize the shrink command", err)
	}

	// Add the shrink command and its associated flags to the root command
	rootCmd.AddCommand(shrinkCmd)
}

// cmdValidShrinkArgs will return which flags and sub-commands are valid for dynamic completion for the shrink command
func cmdValidShrinkArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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

// cmdValidateShrinkArgs makes sure that there are no positional arguments provided to the shrink command
func cmdValidateShrinkArgs(cmd *cobra.Command, args []string) error {
	// Make sure we have no positional args
	if err := cobra.NoArgs(cmd, args); err != nil {
		err = fmt.Errorf("fuzz does not accept any positional arguments, only flags and their associated values")
		cmdLogger.Error("Failed to validate args to the shrink command", err)
		return err
	}
	return nil
}

// cmdRunShrink executes the CLI shrink command and navigates through the following possibilities:
// #1: We will search for either a custom config file (via --config) or the default (medusa.json).
// If we find it, read it. If we can't read it, throw an error.
// #2: If a custom file was provided (--config was used), and we can't find the file, throw an error.
// #3: If medusa.json can't be found, use the default project configuration.
func cmdRunShrink(cmd *cobra.Command, args []string) error {
	var projectConfig *config.ProjectConfig

	// Check to see if --config flag was used and store the value of --config flag
	configFlagUsed := cmd.Flags().Changed("config")
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		cmdLogger.Error("Failed to run the shrink command", err)
		return err
	}

	// If --config was not used, look for `medusa.json` in the current work directory
	if !configFlagUsed {
		workingDirectory, err := os.Getwd()
		if err != nil {
			cmdLogger.Error("Failed to run the shrink command", err)
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
		// Use the default compilation platform if the config file doesn't specify one
		projectConfig, err = config.ReadProjectConfigFromFile(configPath, DefaultCompilationPlatform)
		if err != nil {
			cmdLogger.Error("Failed to run the shrink command", err)
			return err
		}
	}

	// Possibility #2: If the --config flag was used, and we couldn't find the file, we'll throw an error
	if configFlagUsed && existenceError != nil {
		cmdLogger.Error("Failed to run the shrink command", err)
		return existenceError
	}

	// Possibility #3: --config flag was not used and medusa.json was not found, so use the default project config
	if !configFlagUsed && existenceError != nil {
		cmdLogger.Warn(fmt.Sprintf("Unable to find the config file at %v, will use the default project configuration for the "+
			"%v compilation platform instead", configPath, DefaultCompilationPlatform))

		projectConfig, err = config.GetDefaultProjectConfig(DefaultCompilationPlatform)
		if err != nil {
			cmdLogger.Error("Failed to run the shrink command", err)
			return err
		}
	}

	// Update the project configuration given whatever flags were set using the CLI
	err = updateProjectConfigWithShrinkFlags(cmd, projectConfig)
	if err != nil {
		cmdLogger.Error("Failed to run the shrink command", err)
		return err
	}

	// Change our working directory to the parent directory of the project configuration file
	// This is important as when we compile for a given platform, the paths may be relative to wherever the
	// configuration is supplied from. Providing a file path explicitly is optional anyways, so we _should_
	// be in the config directory when running this.
	err = os.Chdir(filepath.Dir(configPath))
	if err != nil {
		cmdLogger.Error("Failed to run the shrink command", err)
		return err
	}

	if !projectConfig.Fuzzing.CoverageEnabled {
		cmdLogger.Warn("Disabling coverage may limit efficacy of fuzzing. Consider enabling coverage for better results.")
	}

	// Create our fuzzing
	fuzzer, fuzzErr := fuzzing.NewFuzzer(*projectConfig)
	if fuzzErr != nil {
		return exitcodes.NewErrorWithExitCode(fuzzErr, exitcodes.ExitCodeHandledError)
	}

	// Stop our fuzzing on keyboard interrupts
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		fuzzer.Stop()
	}()

	// Start the fuzzing process with our cancellable context.
	txFile, err := cmd.Flags().GetString("transaction-file")
	if txFile == "" {
		return fmt.Errorf("failed to get transaction file: %v", err)
	}
	fuzzErr = fuzzer.Shrink(txFile)

	return fuzzErr
}
