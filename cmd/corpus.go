package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/crytic/medusa/fuzzing"
	"github.com/crytic/medusa/fuzzing/config"
	"github.com/crytic/medusa/logging/colors"
	"github.com/spf13/cobra"
)

// corpusCmd represents the corpus command group
var corpusCmd = &cobra.Command{
	Use:   "corpus",
	Short: "Manage the fuzzing corpus",
	Long:  `Commands for managing the fuzzing corpus, including cleaning invalid sequences.`,
}

// corpusCleanCmd represents the corpus clean subcommand
var corpusCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove invalid sequences from the corpus",
	Long: `Validates each call sequence in the corpus by attempting to execute it on a test chain.
Sequences that fail (due to contract changes, ABI mismatches, or execution errors) are removed from disk.

This command is useful after refactoring contracts when the corpus contains many invalid sequences.`,
	RunE:          cmdRunCorpusClean,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// corpusCleanDryRun indicates whether to perform a dry run (report but don't delete)
var corpusCleanDryRun bool

// corpusCleanConfigPath is the path to the config file
var corpusCleanConfigPath string

func init() {
	// Add flags to corpus clean command
	corpusCleanCmd.Flags().BoolVar(
		&corpusCleanDryRun,
		"dry-run",
		false,
		"report invalid sequences without deleting them",
	)
	corpusCleanCmd.Flags().StringVar(
		&corpusCleanConfigPath,
		"config",
		"",
		"path to config file (default: medusa.json in current directory)",
	)

	// Add subcommands to corpus command
	corpusCmd.AddCommand(corpusCleanCmd)

	// Add corpus command to root
	rootCmd.AddCommand(corpusCmd)
}

// cmdRunCorpusClean executes the corpus clean command
func cmdRunCorpusClean(cmd *cobra.Command, args []string) error {
	// Determine config path
	configPath := corpusCleanConfigPath
	if configPath == "" {
		workingDirectory, err := os.Getwd()
		if err != nil {
			cmdLogger.Error("Failed to get working directory", err)
			return err
		}
		configPath = filepath.Join(workingDirectory, DefaultProjectConfigFilename)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); err != nil {
		cmdLogger.Error("Config file not found", err)
		return fmt.Errorf("config file not found at %s", configPath)
	}

	// Read config
	cmdLogger.Info("Reading configuration file at: ", colors.Bold, configPath, colors.Reset)
	projectConfig, err := config.ReadProjectConfigFromFile(configPath, DefaultCompilationPlatform)
	if err != nil {
		cmdLogger.Error("Failed to read config file", err)
		return err
	}

	// Change to config directory
	if err := os.Chdir(filepath.Dir(configPath)); err != nil {
		cmdLogger.Error("Failed to change to config directory", err)
		return err
	}

	// Check if corpus directory is configured
	if projectConfig.Fuzzing.CorpusDirectory == "" {
		cmdLogger.Error("No corpus directory configured", nil)
		return fmt.Errorf("no corpus directory configured in %s", configPath)
	}

	// Check if corpus directory exists
	corpusDir := projectConfig.Fuzzing.CorpusDirectory
	if _, err := os.Stat(corpusDir); os.IsNotExist(err) {
		cmdLogger.Error("Corpus directory does not exist", nil)
		return fmt.Errorf("corpus directory does not exist: %s", corpusDir)
	}

	// Create fuzzer (this handles compilation and contract definitions)
	cmdLogger.Info("Initializing fuzzer...")
	fuzzer, err := fuzzing.NewFuzzer(*projectConfig)
	if err != nil {
		cmdLogger.Error("Failed to initialize fuzzer", err)
		return err
	}

	// Create context with cancellation for interrupt handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		cmdLogger.Info("Interrupted, stopping...")
		cancel()
	}()

	if corpusCleanDryRun {
		cmdLogger.Info("Dry run mode - invalid sequences will be reported but not deleted")
	}

	cmdLogger.Info("Loading and validating corpus from: ", colors.Bold, corpusDir, colors.Reset)

	// Clean the corpus
	result, err := fuzzing.CleanCorpus(ctx, fuzzer, corpusCleanDryRun, cmdLogger)
	if err != nil {
		cmdLogger.Error("Error during corpus cleaning", err)
		return err
	}

	// Report results
	invalidCount := len(result.InvalidSequences)
	cmdLogger.Info(
		"Results: ",
		colors.Bold, result.ValidSequences, colors.Reset, " valid, ",
		colors.Bold, invalidCount, colors.Reset, " invalid out of ",
		colors.Bold, result.TotalSequences, colors.Reset, " total sequences",
	)

	if invalidCount > 0 {
		if corpusCleanDryRun {
			cmdLogger.Info(colors.Bold, invalidCount, colors.Reset, " sequences would be removed (dry run)")
		} else {
			cmdLogger.Info(colors.Bold, invalidCount, colors.Reset, " invalid sequences removed from disk")
		}
	}

	return nil
}
