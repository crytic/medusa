package main

import "github.com/trailofbits/medusa/cmd"

func main() {
	// Run our root CLI command, which contains all underlying command logic and will handle parsing/invocation.
	err := cmd.Execute()

	// Print any error we encountered
	if err != nil {
		// TODO: Replace this when we have an appropriate logger in place.
		panic(err.Error())
	}
}
