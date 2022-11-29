package cmd

import (
	"github.com/spf13/cobra"
	"github.com/trailofbits/medusa/fuzzing/config"
)

// updateCompilationTarget will update the compilation target in the projectConfig if the --target flag is used in the
// command
func updateCompilationTarget(cmd *cobra.Command, projectConfig *config.ProjectConfig) error {
	// If --target was used
	if cmd.Flags().Changed("target") {
		// Get the new target
		newTarget, err := cmd.Flags().GetString("target")
		if err != nil {
			return err
		}

		// Get the platform configuration for the projectConfig
		platformConfig, err := projectConfig.Compilation.GetPlatformConfig()
		if err != nil {
			return err
		}

		// Update the target
		platformConfig.UpdateTarget(newTarget)

		// Update the compilation config
		err = projectConfig.Compilation.UpdatePlatformConfig(platformConfig)
		if err != nil {
			return err
		}
	}
	return nil
}
