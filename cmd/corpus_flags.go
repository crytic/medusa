package cmd

// addCorpusCleanFlags adds flags for the corpus clean subcommand
func addCorpusCleanFlags() error {
	// Prevent alphabetical sorting of usage message
	corpusCleanCmd.Flags().SortFlags = false

	// Config file path
	corpusCleanCmd.Flags().String("config", "",
		"path to config file (default: medusa.json in current directory)")

	return nil
}
