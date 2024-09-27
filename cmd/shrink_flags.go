package cmd

import (
	"fmt"

	"github.com/crytic/medusa/fuzzing/config"
	"github.com/spf13/cobra"
)

// addShrinkFlags adds the various flags for the fuzz command
func addShrinkFlags() error {
	// Get the default project config and throw an error if we cant
	defaultConfig, err := config.GetDefaultProjectConfig(DefaultCompilationPlatform)
	if err != nil {
		return err
	}

	// Prevent alphabetical sorting of usage message
	shrinkCmd.Flags().SortFlags = false

	// Config file
	shrinkCmd.Flags().String("config", "", "path to config file")

	// Compilation Target
	shrinkCmd.Flags().String("compilation-target", "", TargetFlagDescription)

	shrinkCmd.Flags().String("transaction-file", "", "path to transaction file")

	// Timeout
	shrinkCmd.Flags().Int("timeout", 60,
		fmt.Sprintf("number of seconds to spend shrinking testcase(s), default is %v seconds", 60))

	// Target contracts
	shrinkCmd.Flags().StringSlice("target-contracts", []string{},
		fmt.Sprintf("target contracts for fuzz testing (unless a config file is provided, default is %v)", defaultConfig.Fuzzing.TargetContracts))

	// Corpus directory
	shrinkCmd.Flags().String("corpus-dir", "",
		fmt.Sprintf("directory path for corpus items and coverage reports (unless a config file is provided, default is %q)", defaultConfig.Fuzzing.CorpusDirectory))

	// Logging color
	shrinkCmd.Flags().Bool("no-color", false, "disabled colored terminal output")

	return nil
}

// updateProjectConfigWithShrinkFlags will update the given projectConfig with any CLI arguments that were provided to the fuzz command
func updateProjectConfigWithShrinkFlags(cmd *cobra.Command, projectConfig *config.ProjectConfig) error {
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

	// Update timeout
	projectConfig.ShrinkConfig.ShrinkTimeout, err = cmd.Flags().GetInt("timeout")
	fmt.Println("projectConfig.ShrinkConfig.ShrinkTimeout", projectConfig.ShrinkConfig.ShrinkTimeout)
	if err != nil {
		return err
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

	// Update logging color mode
	if cmd.Flags().Changed("no-color") {
		projectConfig.Logging.NoColor, err = cmd.Flags().GetBool("no-color")
		if err != nil {
			return err
		}
	}
	return nil
}
