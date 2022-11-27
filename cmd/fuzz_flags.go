package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/trailofbits/medusa/fuzzing/config"
)

// addFuzzFlags adds the various flags for the fuzz command. It takes in a *config.ProjectConfig that is used to notify
// users of the default values
func addFuzzFlags(config *config.ProjectConfig) {
	// Config file
	fuzzCmd.PersistentFlags().String("config", "", "path to config file")

	// Number of workers
	fuzzCmd.PersistentFlags().Int("workers", 0,
		fmt.Sprintf("number of fuzzer workers (unless a config file is provided, default is %d)", config.Fuzzing.Workers))

	// Timeout
	fuzzCmd.PersistentFlags().Int("timeout", 0,
		fmt.Sprintf("number of seconds to run the fuzzer campaign for (unless a config file is provided, default is %d). 0 means that timeout is not enforced", config.Fuzzing.Timeout))

	// Test limit
	fuzzCmd.PersistentFlags().Uint64("test-limit", 0,
		fmt.Sprintf("number of transactions to test before exiting (unless a config file is provided, default is %d). 0 means that test limit is not enforced", config.Fuzzing.TestLimit))

	// Tx sequence length
	fuzzCmd.PersistentFlags().Int("seq-len", 0,
		fmt.Sprintf("maximum transactions to run in sequence (unless a config file is provided, default is %d)", config.Fuzzing.MaxTxSequenceLength))

	// Deployment order
	fuzzCmd.PersistentFlags().StringSlice("deployment-order", []string{},
		fmt.Sprintf("order in which to deploy target contracts (unless a config file is provided, default is %v)", config.Fuzzing.DeploymentOrder))

	// Corpus directory
	fuzzCmd.PersistentFlags().String("corpus-dir", "",
		fmt.Sprintf("directory path for corpus items and coverage reports (unless a config file is provided, default is %q)", config.Fuzzing.CorpusDirectory))

	// Senders
	fuzzCmd.PersistentFlags().StringSlice("senders", []string{},
		fmt.Sprintf("account address(es) used to send state-changing txns (unless a config file is provided, default addresses are %v)", config.Fuzzing.SenderAddresses))

	// Deployer address
	fuzzCmd.PersistentFlags().String("deployer", "",
		fmt.Sprintf("account address used to deploy contracts (unless a config file is provided, default is %s)", config.Fuzzing.DeployerAddress))

	// Assertion mode
	fuzzCmd.PersistentFlags().Bool("assertion-mode", false,
		fmt.Sprintf("enable assertion mode (unless a config file is provided, default is %t)", config.Fuzzing.Testing.AssertionTesting.Enabled))
}

// updateProjectConfigWithFlags will update the given projectConfig with any CLI arguments that were provided.
func updateProjectConfigWithFlags(cmd *cobra.Command, projectConfig *config.ProjectConfig) error {
	var err error
	// Update number of workers
	if cmd.PersistentFlags().Changed("workers") {
		projectConfig.Fuzzing.Workers, err = cmd.PersistentFlags().GetInt("workers")
		if err != nil {
			return err
		}
	}

	// Update timeout
	if cmd.PersistentFlags().Changed("timeout") {
		projectConfig.Fuzzing.Timeout, err = cmd.PersistentFlags().GetInt("timeout")
		if err != nil {
			return err
		}
	}

	// Update test limit
	if cmd.PersistentFlags().Changed("test-limit") {
		projectConfig.Fuzzing.TestLimit, err = cmd.PersistentFlags().GetUint64("test-limit")
		if err != nil {
			return err
		}
	}

	// Update sequence length
	if cmd.PersistentFlags().Changed("seq-len") {
		projectConfig.Fuzzing.MaxTxSequenceLength, err = cmd.PersistentFlags().GetInt("seq-len")
		if err != nil {
			return err
		}
	}

	// Update deployment order
	if cmd.PersistentFlags().Changed("deployment-order") {
		projectConfig.Fuzzing.DeploymentOrder, err = cmd.PersistentFlags().GetStringSlice("deployment-order")
		if err != nil {
			return err
		}
	}

	// Update corpus directory
	if cmd.PersistentFlags().Changed("corpus-dir") {
		projectConfig.Fuzzing.CorpusDirectory, err = cmd.PersistentFlags().GetString("corpus-dir")
		if err != nil {
			return err
		}
	}

	// Update senders
	if cmd.PersistentFlags().Changed("senders") {
		projectConfig.Fuzzing.SenderAddresses, err = cmd.PersistentFlags().GetStringSlice("senders")
		if err != nil {
			return err
		}
	}

	// Update deployer address
	if cmd.PersistentFlags().Changed("deployer") {
		projectConfig.Fuzzing.DeployerAddress, err = cmd.PersistentFlags().GetString("deployer")
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
