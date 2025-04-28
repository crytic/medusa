package main

import (
	"fmt"
	"os"
	
	"github.com/crytic/medusa/cmd"
	"github.com/crytic/medusa/cmd/exitcodes"
)

func main() {
	// Run the main Medusa CLI command with stylus build tag activated
	err := cmd.Execute()

	// Obtain the actual error and exit code from the error, if any
	var exitCode int
	err, exitCode = exitcodes.GetInnerErrorAndExitCode(err)

	// If we have an error, print it
	if err != nil && exitCode != exitcodes.ExitCodeHandledError {
		fmt.Println(err)
	}

	// If we have a non-success exit code, exit with it
	if exitCode != exitcodes.ExitCodeSuccess {
		os.Exit(exitCode)
	}
}