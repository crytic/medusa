package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/crytic/medusa/logging/colors"

	"github.com/crytic/medusa/compilation"
	"github.com/crytic/medusa/fuzzing/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Get supported platforms for customized static completions of "init" flag `$ medusa init <tab> <tab>`
// and to cache supported platforms for CLI arguments validation
var supportedPlatforms = compilation.GetSupportedCompilationPlatforms()

// initCmd represents the command provider for init
var initCmd = &cobra.Command{
	Use:               "init [platform]",
	Short:             "Initializes a project configuration",
	Long:              `Initializes a project configuration`,
	Args:              cmdValidateInitArgs,
	ValidArgsFunction: cmdValidInitArgs,
	RunE:              cmdRunInit,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

func init() {
	// Add flags to init command
	err := addInitFlags()
	if err != nil {
		cmdLogger.Panic("Failed to initialize the init command", err)
	}

	// Add the init command and its associated flags to the root command
	rootCmd.AddCommand(initCmd)
}

// cmdValidInitArgs will return which flags and sub-commands are valid for dynamic completion for the init command
func cmdValidInitArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Gather a list of flags that are available to be used in the current command but have not been used yet
	var unusedFlags []string

	// Examine all the flags, and add any flags that have not been set in the current command line
	// to a list of unused flags
	flagUsed := false
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if !flag.Changed {
			// When adding a flag to a command, include the "--" prefix to indicate that it is a flag
			// and not a positional argument. Additionally, when the user presses the TAB key twice after typing
			// a flag name, the "--" prefix will appear again, indicating that more flags are available and that
			// none of the arguments are positional.
			unusedFlags = append(unusedFlags, "--"+flag.Name)
		} else {
			// If any flag has been used, set flag used to true. This will be used later in the function.
			flagUsed = true
		}
	})

	// If a default platform is not specified, add a list of available platforms to the list of unused flags.
	// If any flag is used, then we can assume that the default platform is used so we don't need to add supported platforms
	if len(args) == 0 && !flagUsed {
		unusedFlags = append(unusedFlags, supportedPlatforms...)
	}

	// Provide a list of flags that can be used in the current command (but have not been used yet)
	// for autocompletion suggestions
	return unusedFlags, cobra.ShellCompDirectiveNoFileComp
}

// cmdValidateInitArgs validates CLI arguments
func cmdValidateInitArgs(cmd *cobra.Command, args []string) error {
	// Make sure we have no more than 1 arg
	if err := cobra.RangeArgs(0, 1)(cmd, args); err != nil {
		err = fmt.Errorf("init accepts at most 1 platform argument (options: %s). "+
			"default platform is %v\n", strings.Join(supportedPlatforms, ", "), DefaultCompilationPlatform)
		cmdLogger.Error("Failed to validate args to the init command", err)
		return err
	}

	// Ensure the optional provided argument refers to a supported platform
	if len(args) == 1 && !compilation.IsSupportedCompilationPlatform(args[0]) {
		err := fmt.Errorf("init was provided invalid platform argument '%s' (options: %s)", args[0], strings.Join(supportedPlatforms, ", "))
		cmdLogger.Error("Failed to validate args to the init command", err)
		return err
	}

	return nil
}

// cmdRunInit executes the init CLI command and updates the project configuration with any flags
func cmdRunInit(cmd *cobra.Command, args []string) error {
	// Check to see if --out flag was used and store the value of --out flag
	outputFlagUsed := cmd.Flags().Changed("out")
	outputPath, err := cmd.Flags().GetString("out")
	if err != nil {
		cmdLogger.Error("Failed to run the init command", err)
		return err
	}
	// If we weren't provided an output path (flag was not used), we use our working directory
	if !outputFlagUsed {
		workingDirectory, err := os.Getwd()
		if err != nil {
			cmdLogger.Error("Failed to run the init command", err)
			return err
		}
		outputPath = filepath.Join(workingDirectory, DefaultProjectConfigFilename)
	}

	// By default, projectConfig will be the default project config for the DefaultCompilationPlatform
	projectConfig, err := config.GetDefaultProjectConfig(DefaultCompilationPlatform)
	if err != nil {
		cmdLogger.Error("Failed to run the init command", err)
		return err
	}

	// If a platform is provided (and it is not the default), then the projectConfig will be the default project config
	// for that specific compilation platform
	if len(args) == 1 && args[0] != DefaultCompilationPlatform {
		projectConfig, err = config.GetDefaultProjectConfig(args[0])
		if err != nil {
			cmdLogger.Error("Failed to run the init command", err)

			return err
		}
	}

	// Update the project configuration given whatever flags were set using the CLI
	err = updateProjectConfigWithInitFlags(cmd, projectConfig)
	if err != nil {
		cmdLogger.Error("Failed to run the init command", err)
		return err
	}

	if _, err = os.Stat(outputPath); err == nil {
		// Prompt user for overwrite confirmation
		fmt.Print("The file already exists. Overwrite? (y/n): ")
		var response string
		if _, err := fmt.Scan(&response); err != nil {
			// Handle the error (e.g., log it, return an error)
			cmdLogger.Error("Failed to scan input", err)
			return err
		}

		if response != "y" && response != "Y" {
			fmt.Println("Operation canceled.")
			return nil
		}

	}

	// Write our project configuration
	err = projectConfig.WriteToFile(outputPath)
	if err != nil {
		cmdLogger.Error("Failed to run the init command", err)
		return err
	}

	// Print a success message
	if absoluteOutputPath, err := filepath.Abs(outputPath); err == nil {
		outputPath = absoluteOutputPath
	}
	cmdLogger.Info("Project configuration successfully output to: ", colors.Bold, outputPath, colors.Reset)
	return nil
}
