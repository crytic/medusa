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

	// Compilation Target
	fuzzCmd.Flags().String("compilation-target", "", TargetFlagDescription)

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

	// Target contracts
	fuzzCmd.Flags().StringSlice("target-contracts", []string{},
		fmt.Sprintf("target contracts for fuzz testing (unless a config file is provided, default is %v)", defaultConfig.Fuzzing.TargetContracts))

	// Corpus directory
	fuzzCmd.Flags().String("corpus-dir", "",
		fmt.Sprintf("directory path for corpus items and coverage reports (unless a config file is provided, default is %q)", defaultConfig.Fuzzing.CorpusDirectory))

	// Senders
	fuzzCmd.Flags().StringSlice("senders", []string{},
		"account address(es) used to send state-changing txns")

	// Deployer address
	fuzzCmd.Flags().String("deployer", "",
		"account address used to deploy contracts")

	// Logging color
	fuzzCmd.Flags().Bool("no-color", false, "disables colored terminal output")

	// Enable stop on failed test
	fuzzCmd.Flags().Bool("fail-fast", false, "enables stop on failed test")

	// Exploration mode
	fuzzCmd.Flags().Bool("explore", false, "enables exploration mode")

	// Run slither while still trying to use the cache
	fuzzCmd.Flags().Bool("use-slither", false, "runs slither and use the current cached results")

	// Run slither and overwrite the cache
	fuzzCmd.Flags().Bool("use-slither-force", false, "runs slither and overwrite the cached results")

	// RPC url
	fuzzCmd.Flags().String("rpc-url", "", "RPC URL to fetch contracts over")

	// RPC block
	fuzzCmd.Flags().Uint64("rpc-block", 0, "block number to use when fetching contracts over RPC")

	// Verbosity levels (-v, -vv, -vvv)
	fuzzCmd.Flags().CountP("verbosity", "v", "increase verbosity level (can be used multiple times: -v, -vv, -vvv)")
	return nil
}

// updateProjectConfigWithFuzzFlags will update the given projectConfig with any CLI arguments that were provided to the fuzz command
func updateProjectConfigWithFuzzFlags(cmd *cobra.Command, projectConfig *config.ProjectConfig) error {
	var err error

	// If --compilation-target was used
	if cmd.Flags().Changed("compilation-target") {
		// Get the new target
		newTarget, err := cmd.Flags().GetString("compilation-target")
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

	// Update target contracts
	if cmd.Flags().Changed("target-contracts") {
		projectConfig.Fuzzing.TargetContracts, err = cmd.Flags().GetStringSlice("target-contracts")
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

	// Update logging color mode
	if cmd.Flags().Changed("no-color") {
		projectConfig.Logging.NoColor, err = cmd.Flags().GetBool("no-color")
		if err != nil {
			return err
		}
	}

	// Update stop on failed test feature
	if cmd.Flags().Changed("fail-fast") {
		projectConfig.Fuzzing.Testing.StopOnFailedTest, err = cmd.Flags().GetBool("fail-fast")
		if err != nil {
			return err
		}
	}

	// Update configuration to exploration mode
	if cmd.Flags().Changed("explore") {
		explore, err := cmd.Flags().GetBool("explore")
		if err != nil {
			return err
		}
		if explore {
			projectConfig.Fuzzing.Testing.StopOnFailedTest = false
			projectConfig.Fuzzing.Testing.StopOnNoTests = false
			projectConfig.Fuzzing.Testing.AssertionTesting.Enabled = false
			projectConfig.Fuzzing.Testing.PropertyTesting.Enabled = false
			projectConfig.Fuzzing.Testing.OptimizationTesting.Enabled = false
		}
	}

	// Update configuration to run slither while using current cache
	if cmd.Flags().Changed("use-slither") {
		projectConfig.Slither.UseSlither, err = cmd.Flags().GetBool("use-slither")
		if err != nil {
			return err
		}
	}

	// Update configuration to run slither and overwrite the current cache
	if cmd.Flags().Changed("use-slither-force") {
		useSlitherForce, err := cmd.Flags().GetBool("use-slither-force")
		if err != nil {
			return err
		}
		if useSlitherForce {
			projectConfig.Slither.UseSlither = true
			projectConfig.Slither.OverwriteCache = true
		}
	}

	// Update RPC url
	if cmd.Flags().Changed("rpc-url") {
		rpcUrl, err := cmd.Flags().GetString("rpc-url")
		if err != nil {
			return err
		}

		// Enable on-chain fuzzing with the given URL
		projectConfig.Fuzzing.TestChainConfig.ForkConfig.ForkModeEnabled = true
		projectConfig.Fuzzing.TestChainConfig.ForkConfig.RpcUrl = rpcUrl
	}

	// Update RPC block
	if cmd.Flags().Changed("rpc-block") {
		projectConfig.Fuzzing.TestChainConfig.ForkConfig.RpcBlock, err = cmd.Flags().GetUint64("rpc-block")
		if err != nil {
			return err
		}
	}

	// Update the verbosity levels
	if cmd.Flags().Changed("verbosity") || cmd.Flags().Changed("v") {
		verbosityCount, err := cmd.Flags().GetCount("verbosity")
		if err != nil {
			return err
		}

		// Map verbosity count to VerbosityLevel enum
		// -v = Verbose (0)
		// -vv = VeryVerbose (1)
		// -vvv = VeryVeryVerbose (2)
		switch {
		case verbosityCount == 1:
			projectConfig.Fuzzing.Testing.Verbosity = config.Verbose
		case verbosityCount == 2:
			projectConfig.Fuzzing.Testing.Verbosity = config.VeryVerbose
		case verbosityCount >= 3:
			projectConfig.Fuzzing.Testing.Verbosity = config.VeryVeryVerbose
		}
	}

	return nil
}
