package cmd

import (
	"github.com/spf13/cobra"
	"github.com/trailofbits/medusa/fuzzing/config"
)

// addInitFlags adds the various flags for the init command
func addInitFlags() error {
	// Output path for configuration
	initCmd.Flags().String("out", "", "output path for the new project configuration file")

	// Target file / directory
	initCmd.Flags().String("target", "", TargetFlagDescription)

	return nil
}

// updateProjectConfigWithInitFlags will update the given projectConfig with any CLI arguments that were provided to the init command
func updateProjectConfigWithInitFlags(cmd *cobra.Command, projectConfig *config.ProjectConfig) error {
	// If --target was used
	if cmd.Flags().Changed("target") {
		// Get the new target
		newTarget, err := cmd.Flags().GetString("target")
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
