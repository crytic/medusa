package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/crytic/medusa/fuzzing"
	"github.com/crytic/medusa/fuzzing/config"
	"github.com/crytic/medusa/fuzzing/corpus"
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

func init() {
	// Add flags
	err := addCorpusCleanFlags()
	if err != nil {
		cmdLogger.Panic("Failed to initialize the corpus command", err)
	}

	// Add subcommands to corpus command
	corpusCmd.AddCommand(corpusCleanCmd)

	// Add corpus command to root
	rootCmd.AddCommand(corpusCmd)
}

// cmdRunCorpusClean executes the corpus clean command
func cmdRunCorpusClean(cmd *cobra.Command, args []string) error {
	// Get config path from flag
	configFlagUsed := cmd.Flags().Changed("config")
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		cmdLogger.Error("Failed to get config flag", err)
		return err
	}

	if !configFlagUsed {
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

	// Create test chain and build deployed contracts map
	cmdLogger.Info("Setting up test chain...")
	testChain, deployedContracts, err := fuzzer.CreateTestChainForCleaning()
	if err != nil {
		cmdLogger.Error("Failed to setup test chain", err)
		return err
	}
	defer testChain.Close()

	// Create and initialize the corpus
	cmdLogger.Info("Creating corpus...")
	fuzzerCorpus, err := corpus.NewCorpus(projectConfig.Fuzzing.CorpusDirectory)
	if err != nil {
		cmdLogger.Error("Failed to create the corpus", err)
		return err
	}
	err = fuzzerCorpus.Initialize(testChain, fuzzer.ContractDefinitions())
	if err != nil {
		cmdLogger.Error("Failed to initialize the corpus", err)
		return err
	}

	cmdLogger.Info("Loading and validating corpus from: ", colors.Bold, corpusDir, colors.Reset)

	// Create cleaner and run
	cleaner := corpus.NewCorpusCleaner(fuzzerCorpus, cmdLogger)
	start := time.Now()
	result, err := cleaner.Clean(ctx, testChain, deployedContracts)
	if err != nil {
		cmdLogger.Error("Error during corpus cleaning", err)
		return err
	}
	cmdLogger.Info("Corpus cleaning completed in ", time.Since(start).Round(time.Second))

	// Report results
	invalidCount := len(result.InvalidSequences)
	cmdLogger.Info(
		"Results: ",
		colors.Bold, result.ValidSequences, colors.Reset, " valid, ",
		colors.Bold, invalidCount, colors.Reset, " invalid out of ",
		colors.Bold, result.TotalSequences, colors.Reset, " total sequences",
	)

	if invalidCount > 0 {
		cmdLogger.Info(colors.Bold, invalidCount, colors.Reset, " invalid sequences removed from disk")
	}

	return nil
}
