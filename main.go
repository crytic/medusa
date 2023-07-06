package main

import (
	"github.com/crytic/medusa/cmd"
	"os"
)

func main() {
	// Run our root CLI command, which contains all underlying command logic and will handle parsing/invocation.
	err := cmd.Execute()

	if err != nil {
		os.Exit(1)
	}
}
