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

// argFuzzConfigPath describes the project configuration path
var argFuzzConfigPath string

// defaultProjectConfig describes the default project configuration for the default compilation platform
var defaultProjectConfig *config.ProjectConfig

// fuzzCmd represents the command provider for fuzzing
var fuzzCmd = &cobra.Command{
	Use:   "fuzz",
	Short: "Starts a fuzzing campaign",
	Long:  `Starts a fuzzing campaign`,
	RunE:  cmdRunFuzz,
}

func init() {
	// Define err here to prevent variable shadowing of defaultProjectConfig
	var err error
	// Get the default project config
	defaultProjectConfig, err = config.GetDefaultProjectConfig(DefaultCompilationPlatform)
	// Throw a panic if we cannot get the defaultProjectConfig
	if err != nil {
		panic(fmt.Sprintf("unable to get default project config for the %s compilation platform", DefaultCompilationPlatform))
	}

	// Add all the flags allowed for the fuzz command
	addFuzzFlags()

	// Add the fuzz command and its associated flags to the root command
	rootCmd.AddCommand(fuzzCmd)
}

// cmdRunFuzz executes the CLI command
func cmdRunFuzz(cmd *cobra.Command, args []string) error {
	// If we weren't provided an input path, we use our working directory
	if argFuzzConfigPath == "" {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return err
		}
		argFuzzConfigPath = filepath.Join(workingDirectory, DefaultProjectConfigFilename)
	}

	// Read our project configuration
	projectConfig, err := config.ReadProjectConfigFromFile(argFuzzConfigPath)
	// If a config file does not exist, then the default config will be used
	if err != nil {
		projectConfig, err = config.GetDefaultProjectConfig(DefaultCompilationPlatform)
		// Throw if GetDefaultProjectConfig doesn't work
		if err != nil {
			return err
		}
	}

	// Update the project configuration given whatever flags were set using the CLI
	err = updateProjectConfigWithCLIArguments(cmd, projectConfig)
	if err != nil {
		return err
	}
	fmt.Printf("new project config is %v\n", projectConfig)

	// Validate the project config for any errors
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
