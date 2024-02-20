package main

import (
	"github.com/crytic/medusa/cmd"
	"github.com/crytic/medusa/cmd/exitcodes"
	"os"
)

func main() {
	// Run our root CLI command, which contains all underlying command logic and will handle parsing/invocation.
	err := cmd.Execute()

	// Determine the exit code from any potential error and exit out.
	os.Exit(exitcodes.GetErrorExitCode(err))
}
