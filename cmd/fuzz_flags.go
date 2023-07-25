package cmd

import (
	"fmt"

	"github.com/crytic/medusa/fuzzing/config"
	"github.com/spf13/cobra"
)

// addFuzzFlags adds the various flags for the fuzz command
func addFuzzFlags() error {
	// Get the default project config and throw an error if we cant
	defaultConfig, err := config.GetDefaultProjectConfig(DefaultCompilationPlatform)
	if err != nil {
		return err
	}

	// Prevent alphabetical sorting of usage message
	fuzzCmd.Flags().SortFlags = false

	// Config file
	fuzzCmd.Flags().String("config", "", "path to config file")

	// Target
	fuzzCmd.Flags().String("target", "", TargetFlagDescription)

	// Number of workers
	fuzzCmd.Flags().Int("workers", 0,
		fmt.Sprintf("number of fuzzer workers (unless a config file is provided, default is %d)", defaultConfig.Fuzzing.Workers))

	// Timeout
	fuzzCmd.Flags().Int("timeout", 0,
		fmt.Sprintf("number of seconds to run the fuzzer campaign for (unless a config file is provided, default is %d). 0 means that timeout is not enforced", defaultConfig.Fuzzing.Timeout))

	// Test limit
	fuzzCmd.Flags().Uint64("test-limit", 0,
		fmt.Sprintf("number of transactions to test before exiting (unless a config file is provided, default is %d). 0 means that test limit is not enforced", defaultConfig.Fuzzing.TestLimit))

	// Tx sequence length
	fuzzCmd.Flags().Int("seq-len", 0,
		fmt.Sprintf("maximum transactions to run in sequence (unless a config file is provided, default is %d)", defaultConfig.Fuzzing.CallSequenceLength))

	// Deployment order
	fuzzCmd.Flags().StringSlice("deployment-order", []string{},
		fmt.Sprintf("order in which to deploy target contracts (unless a config file is provided, default is %v)", defaultConfig.Fuzzing.DeploymentOrder))

	// Corpus directory
	// TODO: Update description when we add "coverage reports" feature
	fuzzCmd.Flags().String("corpus-dir", "",
		fmt.Sprintf("directory path for corpus items (unless a config file is provided, default is %q)", defaultConfig.Fuzzing.CorpusDirectory))

	// Senders
	fuzzCmd.Flags().StringSlice("senders", []string{},
		"account address(es) used to send state-changing txns")

	// Deployer address
	fuzzCmd.Flags().String("deployer", "",
		"account address used to deploy contracts")

	// Assertion mode
	fuzzCmd.Flags().Bool("assertion-mode", false,
		fmt.Sprintf("enable assertion mode (unless a config file is provided, default is %t)", defaultConfig.Fuzzing.Testing.AssertionTesting.Enabled))

	// Optimization mode
	fuzzCmd.Flags().Bool("optimization-mode", false,
		fmt.Sprintf("enable optimization mode (unless a config file is provided, default is %t)", defaultConfig.Fuzzing.Testing.OptimizationTesting.Enabled))

	// Trace all
	fuzzCmd.Flags().Bool("trace-all", false,
		fmt.Sprintf("print the execution trace for every element in a shrunken call sequence instead of only the last element (unless a config file is provided, default is %t)", defaultConfig.Fuzzing.Testing.TraceAll))
	return nil
}

// updateProjectConfigWithFuzzFlags will update the given projectConfig with any CLI arguments that were provided to the fuzz command
func updateProjectConfigWithFuzzFlags(cmd *cobra.Command, projectConfig *config.ProjectConfig) error {
	var err error

	// If --target was used
	if cmd.Flags().Changed("target") {
		// Get the new target
		newTarget, err := cmd.Flags().GetString("target")
		if err != nil {
			return err
		}

		err = projectConfig.Compilation.SetTarget(newTarget)
		if err != nil {
			return err
		}
	}

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

	// Update deployment order
	if cmd.Flags().Changed("deployment-order") {
		projectConfig.Fuzzing.DeploymentOrder, err = cmd.Flags().GetStringSlice("deployment-order")
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

	// Update senders
	if cmd.Flags().Changed("senders") {
		projectConfig.Fuzzing.SenderAddresses, err = cmd.Flags().GetStringSlice("senders")
		if err != nil {
			return err
		}
	}

	// Update deployer address
	if cmd.Flags().Changed("deployer") {
		projectConfig.Fuzzing.DeployerAddress, err = cmd.Flags().GetString("deployer")
		if err != nil {
			return err
		}
	}

	// Update assertion mode enablement
	if cmd.Flags().Changed("assertion-mode") {
		projectConfig.Fuzzing.Testing.AssertionTesting.Enabled, err = cmd.Flags().GetBool("assertion-mode")
		if err != nil {
			return err
		}
	}

	// Update optimization mode enablement
	if cmd.Flags().Changed("optimization-mode") {
		projectConfig.Fuzzing.Testing.OptimizationTesting.Enabled, err = cmd.Flags().GetBool("optimization-mode")
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
	return nil
}
