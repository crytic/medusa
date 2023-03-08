package cmd

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/trailofbits/medusa/log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trailofbits/medusa/compilation"
	"github.com/trailofbits/medusa/fuzzing/config"
)

// initCmd represents the command provider for init
var initCmd = &cobra.Command{
	Use:           "init [platform]",
	Short:         "Initializes a project configuration",
	Long:          `Initializes a project configuration`,
	Args:          cmdValidateInitArgs,
	RunE:          cmdRunInit,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	// Add flags to init command
	err := addInitFlags()
	if err != nil {
		panic(err)
	}

	// Add the init command and its associated flags to the root command
	rootCmd.AddCommand(initCmd)
}

// cmdValidateInitArgs validates CLI arguments
func cmdValidateInitArgs(cmd *cobra.Command, args []string) error {
	// Create logger instance
	// Not handling error because the only way to trigger an error in NewMultiLogger is if you are logging to a file
	logger, _ := log.NewMultiLogger(zerolog.InfoLevel, "", true)

	// Cache supported platforms
	supportedPlatforms := compilation.GetSupportedCompilationPlatforms()

	// Make sure we have no more than 1 arg
	if err := cobra.RangeArgs(0, 1)(cmd, args); err != nil {
		err = errors.Errorf("init accepts at most 1 platform argument (options: %s). "+
			"default platform is %s", strings.Join(supportedPlatforms, ", "), DefaultCompilationPlatform)
		logger.Error("", log.NewFields("error", err))
		return err
	}

	// Ensure the optional provided argument refers to a supported platform
	if len(args) == 1 && !compilation.IsSupportedCompilationPlatform(args[0]) {
		err := errors.Errorf("init was provided invalid platform argument '%s' (options: %s)", args[0], strings.Join(supportedPlatforms, ", "))
		logger.Error("", log.NewFields("error", err))
		return err
	}

	return nil
}

// cmdRunInit executes the init CLI command and updates the project configuration with any flags
func cmdRunInit(cmd *cobra.Command, args []string) error {
	// Create logger instance
	// Not handling error because the only way to trigger an error in NewMultiLogger is if you are logging to a file
	logger, _ := log.NewMultiLogger(zerolog.InfoLevel, "", true)

	// Check to see if --out flag was used and store the value of --out flag
	outputFlagUsed := cmd.Flags().Changed("out")
	outputPath, err := cmd.Flags().GetString("out")
	if err != nil {
		logger.Error("", log.NewFields("error", errors.WithStack(err)))
		return errors.WithStack(err)
	}

	// If we weren't provided an output path (flag was not used), we use our working directory
	if !outputFlagUsed {
		workingDirectory, err := os.Getwd()
		if err != nil {
			logger.Error("", log.NewFields("error", errors.WithStack(err)))
			return errors.WithStack(err)
		}
		outputPath = filepath.Join(workingDirectory, DefaultProjectConfigFilename)
	}

	// By default, projectConfig will be the default project config for the DefaultCompilationPlatform
	projectConfig, err := config.GetDefaultProjectConfig(DefaultCompilationPlatform)
	if err != nil {
		logger.Error("", log.NewFields("error", err))
		return err
	}

	// If a platform is provided (and it is not the default), then the projectConfig will be the default project config
	// for that specific compilation platform
	if len(args) == 1 && args[0] != DefaultCompilationPlatform {
		projectConfig, err = config.GetDefaultProjectConfig(args[0])
		if err != nil {
			logger.Error("", log.NewFields("error", err))
			return err
		}
	}

	// Update the project configuration given whatever flags were set using the CLI
	err = updateProjectConfigWithInitFlags(cmd, projectConfig)
	if err != nil {
		logger.Error("", log.NewFields("error", err))
		return err
	}

	// Write our project configuration
	err = projectConfig.WriteToFile(outputPath)
	if err != nil {
		logger.Error("", log.NewFields("error", err))
		return err
	}

	// Print a success message
	if absoluteOutputPath, err := filepath.Abs(outputPath); err == nil {
		outputPath = absoluteOutputPath
	}

	logger.Info(fmt.Sprintf("Project configuration successfully output to: %s", outputPath), log.NewFields("service", log.CLI_SERVICE))

	return nil
}
