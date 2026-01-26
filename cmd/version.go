package cmd

import (
	"fmt"

	"github.com/crytic/medusa/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command that displays build information.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version and build information",
	Long: `Print detailed version and build information for medusa.

This includes the semantic version, git commit hash, build timestamp,
and Go version used to compile the binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		info := version.GetInfo()
		fmt.Print(info.String())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
