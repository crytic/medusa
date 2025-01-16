package cmd

import (
	"fmt"

	"github.com/crytic/medusa/fuzzing/config"
	"github.com/spf13/cobra"
)

// addFuzzFlags adds the various flags for the fuzz command
func addReplayFlags() error {
	// Get the default project config and throw an error if we cant
	defaultConfig, err := config.GetDefaultProjectConfig(DefaultCompilationPlatform)
	if err != nil {
		return err
	}

	// Prevent alphabetical sorting of usage message
	replayCmd.Flags().SortFlags = false

	// Config file
	replayCmd.Flags().String("config", "", "path to config file")

	// Number of workers
	replayCmd.Flags().Int("workers", 0,
		fmt.Sprintf("number of fuzzer workers (unless a config file is provided, default is %d)", defaultConfig.Fuzzing.Workers))

	// Timeout
	replayCmd.Flags().Int("timeout", 0,
		fmt.Sprintf("number of seconds to run the fuzzer campaign for (unless a config file is provided, default is %d). 0 means that timeout is not enforced", defaultConfig.Fuzzing.Timeout))

	// Test limit
	replayCmd.Flags().Uint64("test-limit", 0,
		fmt.Sprintf("number of transactions to test before exiting (unless a config file is provided, default is %d). 0 means that test limit is not enforced", defaultConfig.Fuzzing.TestLimit))

	// Tx sequence length
	replayCmd.Flags().Int("seq-len", 0,
		fmt.Sprintf("maximum transactions to run in sequence (unless a config file is provided, default is %d)", defaultConfig.Fuzzing.CallSequenceLength))

	// Corpus directory
	replayCmd.Flags().String("corpus-dir", "",
		fmt.Sprintf("directory path for corpus items and coverage reports (unless a config file is provided, default is %q)", defaultConfig.Fuzzing.CorpusDirectory))

	// Trace all
	replayCmd.Flags().Bool("trace-all", false,
		fmt.Sprintf("print the execution trace for every element in a shrunken call sequence instead of only the last element (unless a config file is provided, default is %t)", defaultConfig.Fuzzing.Testing.TraceAll))

	// Logging color
	replayCmd.Flags().Bool("no-color", false, "disabled colored terminal output")

	return nil
}

// updateProjectConfigWithFuzzFlags will update the given projectConfig with any CLI arguments that were provided to the fuzz command
func updateProjectConfigWithReplayFlags(cmd *cobra.Command, projectConfig *config.ProjectConfig) error {
	var err error

	// Update number of workers
	if cmd.Flags().Changed("workers") {
		projectConfig.Fuzzing.Workers, err = cmd.Flags().GetInt("workers")
		if err != nil {
			return err
		}
	}

	// Update timeout
	if cmd.Flags().Changed("timeout") {
		projectConfig.Fuzzing.Timeout, err = cmd.Flags().GetInt("timeout")
		if err != nil {
			return err
		}
	}

	// Update test limit
	if cmd.Flags().Changed("test-limit") {
		projectConfig.Fuzzing.TestLimit, err = cmd.Flags().GetUint64("test-limit")
		if err != nil {
			return err
		}
	}

	// Update sequence length
	if cmd.Flags().Changed("seq-len") {
		projectConfig.Fuzzing.CallSequenceLength, err = cmd.Flags().GetInt("seq-len")
		if err != nil {
			return err
		}
	}

	// Update corpus directory
	if cmd.Flags().Changed("corpus-dir") {
		projectConfig.Fuzzing.CorpusDirectory, err = cmd.Flags().GetString("corpus-dir")
		if err != nil {
			return err
		}
	}

	// Update trace all enablement
	if cmd.Flags().Changed("trace-all") {
		projectConfig.Fuzzing.Testing.TraceAll, err = cmd.Flags().GetBool("trace-all")
		if err != nil {
			return err
		}
	}

	// Update logging color mode
	if cmd.Flags().Changed("no-color") {
		projectConfig.Logging.NoColor, err = cmd.Flags().GetBool("no-color")
		if err != nil {
			return err
		}
	}
	return nil
}
