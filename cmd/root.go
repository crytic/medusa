package cmd

import (
	"github.com/spf13/cobra"
)

// rootCmd represents the root CLI command object which all other commands stem from.
var rootCmd = &cobra.Command{
	Use:   "medusa",
	Short: "A Solidity smart contract fuzzing harness",
	Long: "medusa is a solidity smart contract fuzzing harness",
}

// Execute provides an exportable function to invoke the CLI.
// Returns an error if one was encountered.
func Execute() error {
	return rootCmd.Execute()
}
