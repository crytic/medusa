package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "medusa",
	Short: "A Solidity smart contract fuzzing harness",
	Long: "medusa is a solidity smart contract fuzzing harness",
}

func Execute() error {
	return rootCmd.Execute()
}