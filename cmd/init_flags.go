package cmd

import (
	"github.com/crytic/medusa/fuzzing/config"
	"github.com/spf13/cobra"
)

// addInitFlags adds the various flags for the init command
func addInitFlags() error {
	// Output path for configuration
	initCmd.Flags().String("out", "", "output path for the new project configuration file")

	// Target file / directory for compilation
	initCmd.Flags().String("compilation-target", "", TargetFlagDescription)

	return nil
}

// updateProjectConfigWithInitFlags will update the given projectConfig with any CLI arguments that were provided to the init command
func updateProjectConfigWithInitFlags(cmd *cobra.Command, projectConfig *config.ProjectConfig) error {
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

	return nil
}
