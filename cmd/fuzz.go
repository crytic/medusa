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

// cmdRunFuzz executes the CLI fuzz command and navigates through the following possibilities:
// If the --config flag is used, we will search for the file at the given config location.
// If the flag was not used, we will look for medusa.json in the current working directly
// If the config file is found, we will parse it. If parsing it fails, we throw an error. Otherwise, we override with CLI values
// If the file is not found, we will use the default config and override with CLI values
func cmdRunFuzz(cmd *cobra.Command, args []string) error {
	// Check to see if --config flag was used and store the value of --config flag
	configFlagUsed := cmd.Flags().Changed("config")
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	// If --config was not used, look for `medusa.json` in the current work directory
	if configFlagUsed == false {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return err
		}
		configPath = filepath.Join(workingDirectory, DefaultProjectConfigFilename)
	}

	var projectConfig *config.ProjectConfig

	// Check to see if the file exists at configPath
	_, err = os.Stat(configPath)

	// If the file exists, let us read it
	if err == nil {
		// Read the configuration file
		projectConfig, err = config.ReadProjectConfigFromFile(configPath)
		// There was some kind of error while parsing the config, return it
		if err != nil {
			return err
		}
	} else {
		// Since we can't find the file, we will use the default config for the default compilation platform
		// and notify the user
		fmt.Printf("unable to find the config file at %v. will use the default project configuration for the "+
			"%v compilation platform instead\n", configPath, DefaultCompilationPlatform)

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

	// Change our working directory to the parent directory of the project configuration file
	// This is important as when we compile for a given platform, the paths may be relative to wherever the
	// configuration is supplied from. Providing a file path explicitly is optional anyways, so we _should_
	// be in the config directory when running this.
	err = os.Chdir(filepath.Dir(configPath))
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
