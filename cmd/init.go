package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/trailofbits/medusa/compilation"
	"github.com/trailofbits/medusa/fuzzing/config"
	"os"
	"path/filepath"
	"strings"
)

// argInitOutputPath describes the init output directory parsed from CLI arguments
var argInitOutputPath string

// initCmd represents the command provider for init
var initCmd = &cobra.Command{
	Use:   "init [platform]",
	Short: "Initializes a project configuration",
	Long:  `Initializes a project configuration`,
	Args:  cmdValidateInitArgs,
	RunE:  cmdRunInit,
}

func init() {
	initCmd.PersistentFlags().StringVarP(&argInitOutputPath, "out", "o", "", "output path for the new project configuration")
	rootCmd.AddCommand(initCmd)
}

// cmdValidateInitArgs validates CLI arguments
func cmdValidateInitArgs(cmd *cobra.Command, args []string) error {
	// Validate we have a positional argument to represent our platform
	if err := cobra.ExactArgs(1)(cmd, args); err != nil {
		supportedPlatforms := compilation.GetSupportedCompilationPlatforms()
		return fmt.Errorf("init requires a platform argument (options: %s)", strings.Join(supportedPlatforms, ", "))
	}

	// Ensure the provided argument refers to a supported platform
	if !compilation.IsSupportedCompilationPlatform(args[0]) {
		supportedPlatforms := compilation.GetSupportedCompilationPlatforms()
		return fmt.Errorf("init was provided invalid platform argument '%s' (options: %s)", args[0], strings.Join(supportedPlatforms, ", "))
	}

	return nil
}

// cmdRunInit executes the init CLI command
func cmdRunInit(cmd *cobra.Command, args []string) error {
	// If we weren't provided an output path, we use our working directory
	if argInitOutputPath == "" {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return err
		}
		argInitOutputPath = filepath.Join(workingDirectory, DefaultProjectConfigFilename)
	}

	// Get a default project configuration
	projectConfig, err := config.GetDefaultProjectConfig(args[0])
	if err != nil {
		return err
	}

	// Read our project configuration
	err = projectConfig.WriteToFile(argInitOutputPath)
	if err != nil {
		return err
	}

	// Print a success message
	if absoluteOutputPath, err := filepath.Abs(argInitOutputPath); err == nil {
		argInitOutputPath = absoluteOutputPath
	}
	fmt.Printf("Project configuration successfully output to: %s\n", argInitOutputPath)
	return nil
}
