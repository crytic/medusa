package main

import (
	"fmt"
	"github.com/crytic/medusa/cmd"
	"os"
)

func main() {
	// Run our root CLI command, which contains all underlying command logic and will handle parsing/invocation.
	err := cmd.Execute()

	// Print any error we encountered
	if err != nil {
		// TODO: Replace this when we have an appropriate logger in place.
		fmt.Printf("ERROR:\n%s", err.Error())
		os.Exit(1)
	}
}
