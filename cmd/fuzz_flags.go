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
	fuzzCmd.LocalFlags().String("config", "", "path to config file")

	// Number of workers
	fuzzCmd.LocalFlags().Int("workers", 0,
		fmt.Sprintf("number of fuzzer workers (unless a config file is provided, default is %d)", config.Fuzzing.Workers))

	// Timeout
	fuzzCmd.LocalFlags().Int("timeout", 0,
		fmt.Sprintf("number of seconds to run the fuzzer campaign for (unless a config file is provided, default is %d). 0 means that timeout is not enforced", config.Fuzzing.Timeout))

	// Test limit
	fuzzCmd.LocalFlags().Uint64("test-limit", 0,
		fmt.Sprintf("number of transactions to test before exiting (unless a config file is provided, default is %d). 0 means that test limit is not enforced", config.Fuzzing.TestLimit))

	// Tx sequence length
	fuzzCmd.LocalFlags().Int("seq-len", 0,
		fmt.Sprintf("maximum transactions to run in sequence (unless a config file is provided, default is %d)", config.Fuzzing.MaxTxSequenceLength))

	// Deployment order
	fuzzCmd.LocalFlags().StringSlice("deployment-order", []string{},
		fmt.Sprintf("order in which to deploy target contracts (unless a config file is provided, default is %v)", config.Fuzzing.DeploymentOrder))

	// Corpus directory
	fuzzCmd.LocalFlags().String("corpus-dir", "",
		fmt.Sprintf("directory path for corpus items and coverage reports (unless a config file is provided, default is %q)", config.Fuzzing.CorpusDirectory))

	// Senders
	fuzzCmd.LocalFlags().StringSlice("senders", []string{},
		fmt.Sprintf("account address(es) used to send state-changing txns (unless a config file is provided, default addresses are %v)", config.Fuzzing.SenderAddresses))

	// Deployer address
	fuzzCmd.LocalFlags().String("deployer", "",
		fmt.Sprintf("account address used to deploy contracts (unless a config file is provided, default is %s)", config.Fuzzing.DeployerAddress))

	// Assertion mode
	fuzzCmd.LocalFlags().Bool("assertion-mode", false,
		fmt.Sprintf("enable assertion mode (unless a config file is provided, default is %t)", config.Fuzzing.Testing.AssertionTesting.Enabled))
}

// updateProjectConfigWithFlags will update the given projectConfig with any CLI arguments that were provided.
func updateProjectConfigWithFlags(cmd *cobra.Command, projectConfig *config.ProjectConfig) error {
	var err error
	// Update number of workers
	if cmd.LocalFlags().Changed("workers") {
		projectConfig.Fuzzing.Workers, err = cmd.LocalFlags().GetInt("workers")
		if err != nil {
			return err
		}
	}

	// Update timeout
	if cmd.LocalFlags().Changed("timeout") {
		projectConfig.Fuzzing.Timeout, err = cmd.LocalFlags().GetInt("timeout")
		if err != nil {
			return err
		}
	}

	// Update test limit
	if cmd.LocalFlags().Changed("test-limit") {
		projectConfig.Fuzzing.TestLimit, err = cmd.LocalFlags().GetUint64("test-limit")
		if err != nil {
			return err
		}
	}

	// Update sequence length
	if cmd.LocalFlags().Changed("seq-len") {
		projectConfig.Fuzzing.MaxTxSequenceLength, err = cmd.LocalFlags().GetInt("seq-len")
		if err != nil {
			return err
		}
	}

	// Update deployment order
	if cmd.LocalFlags().Changed("deployment-order") {
		projectConfig.Fuzzing.DeploymentOrder, err = cmd.LocalFlags().GetStringSlice("deployment-order")
		if err != nil {
			return err
		}
	}

	// Update corpus directory
	if cmd.LocalFlags().Changed("corpus-dir") {
		projectConfig.Fuzzing.CorpusDirectory, err = cmd.LocalFlags().GetString("corpus-dir")
		if err != nil {
			return err
		}
	}

	// Update senders
	if cmd.LocalFlags().Changed("senders") {
		projectConfig.Fuzzing.SenderAddresses, err = cmd.LocalFlags().GetStringSlice("senders")
		if err != nil {
			return err
		}
	}

	// Update deployer address
	if cmd.LocalFlags().Changed("deployer") {
		projectConfig.Fuzzing.DeployerAddress, err = cmd.LocalFlags().GetString("deployer")
		if err != nil {
			return err
		}
	}

	// Update assertion mode enablement
	if cmd.LocalFlags().Changed("assertion-mode") {
		projectConfig.Fuzzing.Testing.AssertionTesting.Enabled, err = cmd.LocalFlags().GetBool("assertion-mode")
		if err != nil {
			return err
		}
	}

	return nil
}
