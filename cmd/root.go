package cmd

import (
	"github.com/crytic/medusa/logging"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"io"
)

const version = "0.1.1"

// rootCmd represents the root CLI command object which all other commands stem from.
var rootCmd = &cobra.Command{
	Use:     "medusa",
	Version: version,
	Short:   "A Solidity smart contract fuzzing harness",
	Long:    "medusa is a solidity smart contract fuzzing harness",
}

// cmdLogger is the logger that will be used for the cmd package
var cmdLogger = logging.NewLogger(zerolog.InfoLevel, true, make([]io.Writer, 0)...)

// Execute provides an exportable function to invoke the CLI.
// Returns an error if one was encountered.
func Execute() error {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	return rootCmd.Execute()
}
