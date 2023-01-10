package cmd

import (
	"fmt"
	"github.com/spf13/pflag"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trailofbits/medusa/compilation"
	"github.com/trailofbits/medusa/fuzzing/config"
)

// Get supported platforms for customized static completions of "init" flag `$ medusa init <tab> <tab>`
// and to cache supported platforms for CLI arguments validation
var supportedPlatforms = compilation.GetSupportedCompilationPlatforms()

// initCmd represents the command provider for init
var initCmd = &cobra.Command{
	Use:   "init [platform]",
	Short: "Initializes a project configuration",
	Long:  `Initializes a project configuration`,
	Args:  cmdValidateInitArgs,
	RunE:  cmdRunInit,
	// Run dynamic completion of nouns
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Collect list of flags that were not changed
		var unusedFlags []string

		// Visit all the flags and if the flag is not changed append to unusedFlags list
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			if !flag.Changed {
				unusedFlags = append(unusedFlags, flag.Name)
			}
		})

		// When default platform is not provided append unusedFlags with platforms
		// When a --target or --out is provided - assume the default platform is used
		if len(args) == 0 && !(cmd.Flag("target").Changed || cmd.Flag("out").Changed) {
			unusedFlags = append(unusedFlags, supportedPlatforms...)
			//return unusedFlags, cobra.ShellCompDirectiveNoFileComp
		}

		// Return unused flags for autocompletion
		return unusedFlags, cobra.ShellCompDirectiveNoFileComp
	},
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
	// Make sure we have no more than 1 arg
	if err := cobra.RangeArgs(0, 1)(cmd, args); err != nil {
		return fmt.Errorf("init accepts at most 1 platform argument (options: %s). "+
			"default platform is %v\n", strings.Join(supportedPlatforms, ", "), DefaultCompilationPlatform)
	}

	// Ensure the optional provided argument refers to a supported platform
	if len(args) == 1 && !compilation.IsSupportedCompilationPlatform(args[0]) {
		return fmt.Errorf("init was provided invalid platform argument '%s' (options: %s)", args[0], strings.Join(supportedPlatforms, ", "))
	}

	return nil
}

// cmdRunInit executes the init CLI command and updates the project configuration with any flags
func cmdRunInit(cmd *cobra.Command, args []string) error {
	// Check to see if --out flag was used and store the value of --out flag
	outputFlagUsed := cmd.Flags().Changed("out")
	outputPath, err := cmd.Flags().GetString("out")
	if err != nil {
		return err
	}
	// If we weren't provided an output path (flag was not used), we use our working directory
	if !outputFlagUsed {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return err
		}
		outputPath = filepath.Join(workingDirectory, DefaultProjectConfigFilename)
	}

	// By default, projectConfig will be the default project config for the DefaultCompilationPlatform
	projectConfig, err := config.GetDefaultProjectConfig(DefaultCompilationPlatform)
	if err != nil {
		return err
	}

	// If a platform is provided (and it is not the default), then the projectConfig will be the default project config
	// for that specific compilation platform
	if len(args) == 1 && args[0] != DefaultCompilationPlatform {
		projectConfig, err = config.GetDefaultProjectConfig(args[0])
		if err != nil {
			return err
		}
	}

	// Update the project configuration given whatever flags were set using the CLI
	err = updateProjectConfigWithInitFlags(cmd, projectConfig)
	if err != nil {
		return err
	}

	// Write our project configuration
	err = projectConfig.WriteToFile(outputPath)
	if err != nil {
		return err
	}

	// Print a success message
	if absoluteOutputPath, err := filepath.Abs(outputPath); err == nil {
		outputPath = absoluteOutputPath
	}
	fmt.Printf("Project configuration successfully output to: %s\n", outputPath)
	return nil
}
