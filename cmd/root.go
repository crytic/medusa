package cmd

import (
	"github.com/crytic/medusa/logging"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"os"
)

const version = "0.1.6"

// rootCmd represents the root CLI command object which all other commands stem from.
var rootCmd = &cobra.Command{
	Use:     "medusa",
	Version: version,
	Short:   "A Solidity smart contract fuzzing harness",
	Long:    "medusa is a solidity smart contract fuzzing harness",
}

// cmdLogger is the logger that will be used for the cmd package
var cmdLogger = logging.NewLogger(zerolog.InfoLevel)

// Execute provides an exportable function to invoke the CLI. Returns an error if one was encountered.
func Execute() error {
	// Add stdout as an unstructured, colorized output stream for the command logger
	cmdLogger.AddWriter(os.Stdout, logging.UNSTRUCTURED, true)

	rootCmd.CompletionOptions.DisableDefaultCmd = true
	return rootCmd.Execute()
}
