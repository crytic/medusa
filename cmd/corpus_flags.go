package cmd

import "github.com/spf13/cobra"

// addCorpusCleanFlags adds flags for the corpus clean subcommand
func addCorpusCleanFlags() error {
	return addCorpusCleanFlagsToCommand(corpusCleanCmd)
}

// addCorpusCleanFlagsToCommand adds flags for the corpus clean subcommand to the provided command.
func addCorpusCleanFlagsToCommand(cmd *cobra.Command) error {
	// Prevent alphabetical sorting of usage message
	cmd.Flags().SortFlags = false

	// Config file path
	cmd.Flags().String("config", "",
		"path to config file (default: medusa.json in current directory)")

	return nil
}
