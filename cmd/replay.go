package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/crytic/medusa/cmd/exitcodes"
	"github.com/crytic/medusa/logging"
	"github.com/crytic/medusa/logging/colors"

	"github.com/crytic/medusa/fuzzing"
	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/fuzzing/config"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/spf13/cobra"
)

// replayCmd represents the command provider for fuzzing
var replayCmd = &cobra.Command{
	Use:               "replay",
	Short:             "Replay a fuzzing campaign",
	Long:              `Replay a fuzzing campaign`,
	Args:              cmdValidateFuzzArgs,
	ValidArgsFunction: cmdValidFuzzArgs,
	RunE:              cmdRunReplay,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

func init() {
	// Add all the flags allowed for the fuzz command
	err := addReplayFlags()
	if err != nil {
		cmdLogger.Panic("Failed to initialize the fuzz command", err)
	}

	// Add the fuzz command and its associated flags to the root command
	rootCmd.AddCommand(replayCmd)
}

// cmdRunReplay executes the CLI fuzz command and navigates through the following possibilities:
// #1: We will search for either a custom config file (via --config) or the default (medusa.json).
// If we find it, read it. If we can't read it, throw an error.
// #2: If a custom file was provided (--config was used), and we can't find the file, throw an error.
// #3: If medusa.json can't be found, use the default project configuration.
func cmdRunReplay(cmd *cobra.Command, args []string) error {
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
		// Use the default compilation platform if the config file doesn't specify one
		projectConfig, err = config.ReadProjectConfigFromFile(configPath, DefaultCompilationPlatform)
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

	if !projectConfig.Fuzzing.CoverageEnabled {
		cmdLogger.Warn("Disabling coverage may limit efficacy of fuzzing. Consider enabling coverage for better results.")
	}

	// Create our fuzzing
	fuzzer, fuzzErr := fuzzing.NewFuzzer(*projectConfig)
	if fuzzErr != nil {
		return exitcodes.NewErrorWithExitCode(fuzzErr, exitcodes.ExitCodeHandledError)
	}

	// Stop our fuzzing on keyboard interrupts
	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt)
	// go func() {
	// 	<-c
	// 	fuzzer.Stop()
	// }()

	// // Start the fuzzing process with our cancellable context.
	// fuzzErr = fuzzer.Start()
	chain, err := fuzzer.CreateTestChainWithAllocFile()

	if err != nil {
		return err
	}
	// Read the file data.
	b, err := os.ReadFile("crash.json")
	if err != nil {
		return err
	}

	fmt.Println("Loaded data", string(b))
	// Parse the call sequence data.
	var sequence calls.CallSequence
	err = json.Unmarshal(b, &sequence)
	if err != nil {
		return err
	}

	fmt.Println("Loaded sequence", len(sequence))

	// fetchElementFunc := func(currentIndex int) (*calls.CallSequenceElement, error) {
	// 	// If we are at the end of our sequence, return nil indicating we should stop executing.
	// 	if currentIndex >= len(sequence) {
	// 		return nil, nil
	// 	}

	// 	// If we are deploying a contract and not targeting one with this call, there should be no work to do.
	// 	currentSequenceElement := sequence[currentIndex]

	// 	return currentSequenceElement, nil

	// }

	executed, err := calls.ExecuteCallSequenceWithExecutionTracer(chain, contracts.Contracts{}, sequence, true)
	if err != nil {

		logging.GlobalLogger.Panic(err)
	}
	for _, call := range executed {
		if call.ExecutionTrace != nil {
			logging.GlobalLogger.Info(call.ExecutionTrace.Log())
		} else {
			logging.GlobalLogger.Info("No trace for call")
		}
	}

	if fuzzErr != nil {
		return exitcodes.NewErrorWithExitCode(fuzzErr, exitcodes.ExitCodeHandledError)
	}

	// If we have no error and failed test cases, we'll want to return a special exit code
	if fuzzErr == nil && len(fuzzer.TestCasesWithStatus(fuzzing.TestCaseStatusFailed)) > 0 {
		return exitcodes.NewErrorWithExitCode(fuzzErr, exitcodes.ExitCodeTestFailed)
	}

	return fuzzErr
}
