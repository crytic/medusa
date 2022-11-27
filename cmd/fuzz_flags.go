package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/trailofbits/medusa/fuzzing/config"
)

// addFuzzFlags is a simple helper function that adds the various positional arguments / flags for the fuzz command
func addFuzzFlags() {
	// Config file location
	fuzzCmd.PersistentFlags().StringVar(&argFuzzConfigPath, "config", "", "path to config file")
	// Number of workers
	fuzzCmd.PersistentFlags().IntVar(&defaultProjectConfig.Fuzzing.Workers, "workers", 0,
		fmt.Sprintf("number of fuzzer workers (default is %d unless a config file is provided)", defaultProjectConfig.Fuzzing.Workers))
	// Test limit
	fuzzCmd.PersistentFlags().Uint64Var(&defaultProjectConfig.Fuzzing.TestLimit, "test-limit", 0,
		fmt.Sprintf("number of transactions to test before exiting (default is %d unless a config file is provided). 0 means that test limit is not enforced", defaultProjectConfig.Fuzzing.TestLimit))
	// Tx sequence length
	fuzzCmd.PersistentFlags().IntVar(&defaultProjectConfig.Fuzzing.MaxTxSequenceLength, "seq-len", 0,
		fmt.Sprintf("maximum transactions to run in sequence (default is %d unless a config file is provided)", defaultProjectConfig.Fuzzing.MaxTxSequenceLength))
	// Corpus directory
	fuzzCmd.PersistentFlags().StringVar(&defaultProjectConfig.Fuzzing.CorpusDirectory, "corpus-dir", "",
		fmt.Sprintf("directory path for corpus items and coverage reports (default is %q unless a config file is provided)", defaultProjectConfig.Fuzzing.CorpusDirectory))
	// Deployment order
	fuzzCmd.PersistentFlags().StringSliceVar(&defaultProjectConfig.Fuzzing.DeploymentOrder, "deployment-order", []string{},
		fmt.Sprintf("order in which to deploy target contracts (default is %v unless a config file is provided)", defaultProjectConfig.Fuzzing.DeploymentOrder))
	// Assertion mode
	fuzzCmd.PersistentFlags().BoolVar(&defaultProjectConfig.Fuzzing.Testing.AssertionTesting.Enabled, "assertion-mode", false,
		fmt.Sprintf("enable assertion mode (default is %t unless a config file is provided)", defaultProjectConfig.Fuzzing.Testing.AssertionTesting.Enabled))
}

// updateProjectConfigWithCLIArguments will update the given projectConfig with any CLI arguments that are provided.
// Note that updates are only made if a CLI argument diverges from its default value
func updateProjectConfigWithCLIArguments(cmd *cobra.Command, projectConfig *config.ProjectConfig) error {
	// Update number of workers
	var err error
	if cmd.Flags().Changed("workers") {
		projectConfig.Fuzzing.Workers, err = cmd.Flags().GetInt("workers")
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
		projectConfig.Fuzzing.MaxTxSequenceLength, err = cmd.Flags().GetInt("seq-len")
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

	// Update deployment order
	if cmd.Flags().Changed("deployment-order") {
		projectConfig.Fuzzing.DeploymentOrder, err = cmd.Flags().GetStringSlice("deployment-order")
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
	return nil
}
