package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/trailofbits/medusa/fuzzing"
	"github.com/trailofbits/medusa/fuzzing/config"
	"os"
	"os/signal"
	"path/filepath"
)

// fuzzCmd represents the command provider for fuzzing
var fuzzCmd = &cobra.Command{
	Use:   "fuzz",
	Short: "Starts a fuzzing campaign",
	Long:  `Starts a fuzzing campaign`,
	Args:  cmdValidateFuzzArgs,
	RunE:  cmdRunFuzz,
}

// cmdValidateFuzzArgs makes sure that there are no positional arguments provided to the fuzz command
func cmdValidateFuzzArgs(cmd *cobra.Command, args []string) error {
	// Make sure we have no positional args
	if err := cobra.NoArgs(cmd, args); err != nil {
		return fmt.Errorf("fuzz does not accept any positional arguments, only flags and their associated values")
	}
	return nil
}

func init() {
	// Get the default project config and throw a panic if we can't
	defaultProjectConfig, err := config.GetDefaultProjectConfig(DefaultCompilationPlatform)
	if err != nil {
		panic(fmt.Sprintf("unable to get default project config for the %s compilation platform", DefaultCompilationPlatform))
	}

	// Prevent lexicographic sorting of flags to maintain parity with order in the config file
	// Note: this is a bit clunky and will have to change if we update the config in any way
	// TODO: Fine with removing it if unnecessary
	fuzzCmd.Flags().SortFlags = false

	// Add all the flags allowed for the fuzz command
	addFuzzFlags(defaultProjectConfig)

	// Add the fuzz command and its associated flags to the root command
	rootCmd.AddCommand(fuzzCmd)
}

// cmdRunFuzz executes the CLI fuzz command with three distinct possibilities:
// First, if a --config file is specified, read that. If we don't find the file, throw an error and exit
// Second, we try for the default file. If we find that, we continue and override with CLI args
// Third, if we don't find the default file, we will use the CLI as the default config
func cmdRunFuzz(cmd *cobra.Command, args []string) error {
	// Check to see if --config flag was used and store the value of --config flag
	changed := cmd.Flags().Changed("config")
	argFuzzConfigPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	// If --config was not used, look for `medusa.json` in the current work directory
	if changed == false {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return err
		}
		argFuzzConfigPath = filepath.Join(workingDirectory, DefaultProjectConfigFilename)
	}

	// Read the configuration file
	projectConfig, err := config.ReadProjectConfigFromFile(argFuzzConfigPath)

	// If we failed to read and the --config parameter was used, that means we did not find the file
	if err != nil && changed == true {
		return err
	}

	// If we failed BUT --config was not used, we should use the default config for the default platform
	if err != nil && changed == false {
		fmt.Printf("unable to find the config file at %v. will use the default project configuration for the "+
			"%v compilation platform instead\n", argFuzzConfigPath, DefaultCompilationPlatform)
		projectConfig, err = config.GetDefaultProjectConfig(DefaultCompilationPlatform)
		if err != nil {
			return err
		}
	}

	// Update the project configuration given whatever flags were set using the CLI
	err = updateProjectConfigWithFlags(cmd, projectConfig)
	if err != nil {
		return err
	}

	// Validate the project config for any errors
	// Note that the CLI arguments have not gone through any previous validation
	err = projectConfig.Validate()
	if err != nil {
		return err
	}

	// Change our working directory to the parent directory of the project configuration file
	// This is important as when we compile for a given platform, the paths may be relative to wherever the
	// configuration is supplied from. Providing a file path explicitly is optional anyways, so we _should_
	// be in the config directory when running this.
	err = os.Chdir(filepath.Dir(argFuzzConfigPath))
	if err != nil {
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

	return err
}
