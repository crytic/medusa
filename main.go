package main

import (
	"fmt"
	"github.com/crytic/medusa/cmd"
	"github.com/crytic/medusa/cmd/exitcodes"
	"os"
)

func main() {
	// Run our root CLI command, which contains all underlying command logic and will handle parsing/invocation.
	err := cmd.Execute()

	// If we have a special (non-zero/success) error code, then exit with it.
	exitCode := exitcodes.GetErrorExitCode(err)
	if exitCode != exitcodes.ExitCodeSuccess {
		fmt.Println(err.Error())
		os.Exit(exitCode)
	}
}
