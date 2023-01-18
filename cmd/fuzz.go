package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trailofbits/medusa/fuzzing"
	"github.com/trailofbits/medusa/fuzzing/config"
	"github.com/trailofbits/medusa/utils"
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
	// Add all the flags allowed for the fuzz command
	err := addFuzzFlags()
	if err != nil {
		panic(err)
	}

	// Add the fuzz command and its associated flags to the root command
	rootCmd.AddCommand(fuzzCmd)
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
		return err
	}

	// If --config was not used, look for `medusa.json` in the current work directory
	if !configFlagUsed {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return err
		}
		configPath = filepath.Join(workingDirectory, DefaultProjectConfigFilename)
	}

	// Check to see if the file exists at configPath
	_, existenceError := os.Stat(configPath)

	// Possibility #1: File was found
	if existenceError == nil {
		// Try to read the configuration file and throw an error if something goes wrong
		projectConfig, err = config.ReadProjectConfigFromFile(configPath)
		if err != nil {
			return err
		}
	}

	// Possibility #2: If the --config flag was used, and we couldn't find the file, we'll throw an error
	if configFlagUsed && existenceError != nil {
		return existenceError
	}

	// Possibility #3: --config flag was not used and medusa.json was not found, so use the default project config
	if !configFlagUsed && existenceError != nil {
		fmt.Printf("unable to find the config file at %v. will use the default project configuration for the "+
			"%v compilation platform instead\n", configPath, DefaultCompilationPlatform)

		projectConfig, err = config.GetDefaultProjectConfig(DefaultCompilationPlatform)
		if err != nil {
			return err
		}
	}

	// Update the project configuration given whatever flags were set using the CLI
	err = updateProjectConfigWithFuzzFlags(cmd, projectConfig)
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

	// Check if any method in the ABI starts with a configured prefix when property testing is enabled.
	if projectConfig.Fuzzing.Testing.PropertyTesting.Enabled {
		// slice that contains `testPrefixes` from the project configuration
		testPrefixes := projectConfig.Fuzzing.Testing.PropertyTesting.TestPrefixes

		// A variable that store the status whether any method starts with appropriate prefix
		contains := false

		// deploymentOrder
		deploymentOrder := projectConfig.Fuzzing.DeploymentOrder
		// Iterate over contracts definitions
		for _, contract := range fuzzer.ContractDefinitions() {
			// Check if the contract name is deployed (set up in the deploymentOrder)
			if utils.Contains(deploymentOrder, contract.Name()) {
				// Iterate over all possible methods in the specific contract
				methods := contract.CompiledContract().Abi.Methods

				for key := range methods {
					// Iterate over all testPrefixes from the configuration files
					// And mark if any of them has appropriate prefix
					for _, prefix := range testPrefixes {
						if strings.HasPrefix(key, prefix) {
							contains = true
							break
						}
					}
				}
			}
		}
		// Return error if no method starts with a prefix from testPrefixes
		if !contains {
			return fmt.Errorf("not found any method with the prefixes [%s] in the ABI",
				strings.Join(testPrefixes, ", "))
		}
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
