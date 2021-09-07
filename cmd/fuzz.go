package cmd

import (
	"github.com/spf13/cobra"
	"github.com/trailofbits/medusa/configs"
	"github.com/trailofbits/medusa/fuzzer"
	"os"
	"os/signal"
	"path"
)

// argFuzzConfigPath describes the project configuration path
var argFuzzConfigPath string

// fuzzCmd represents the command provider for fuzzing
var fuzzCmd = &cobra.Command{
	Use:   "fuzz",
	Short: "Starts a fuzzing campaign",
	Long:  `Starts a fuzzing campaign`,
	RunE: cmdRunFuzz,
}

func init() {
	fuzzCmd.PersistentFlags().StringVarP(&argFuzzConfigPath, "in", "i", "", "project configuration input path")
	rootCmd.AddCommand(fuzzCmd)
}

// cmdRunFuzz executes the CLI command
func cmdRunFuzz(cmd *cobra.Command, args []string) error {
	// If we weren't provided an input path, we use our working directory
	if argFuzzConfigPath == "" {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return err
		}
		argFuzzConfigPath = path.Join(workingDirectory, DefaultProjectConfigFilename)
	}

	// Read our project configuration
	projectConfig, err := configs.ReadProjectConfigFromFile(argFuzzConfigPath)
	if err != nil {
		return err
	}

	// Change our working directory to the parent directory of the project configuration file
	// This is important as when we compile for a given platform, the paths may be relative to wherever the
	// configuration is supplied from. Providing a file path explicitly is optional anyways, so we _should_
	// be in the config directory when running this.
	err = os.Chdir(path.Dir(argFuzzConfigPath))
	if err != nil {
		return err
	}

	// Create our fuzzer
	fuzzer, err := fuzzer.NewFuzzer(*projectConfig)
	if err != nil {
		return err
	}

	// Stop our fuzzer on keyboard interrupts
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		fuzzer.Stop()
	}()

	// Start the fuzzing process with our cancellable context.
	err = fuzzer.Start()

	return err
}