package main

import (
	"github.com/trailofbits/medusa/cmd"
	"os"
)

func main() {
	// Create logger
	// Not handling error because the only way to trigger an error in NewMultiLogger is if you are logging to a file
	//logger, _ := log.NewMultiLogger(zerolog.ErrorLevel, "temp", true)
	// Run our root CLI command, which contains all underlying command logic and will handle parsing/invocation.
	err := cmd.Execute()

	// Print any error we encountered
	if err != nil {
		//logger.Error("Encountered error during execution", log.NewFields("error", err))
		os.Exit(1)
	}
}
