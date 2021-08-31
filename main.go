package main

import "medusa/cmd"

func main() {
	// Run our root command, which will do all underlying parsing and invocation.
	err := cmd.Execute()

	// Print any error we encountered
	if err != nil {
		panic(err.Error())
	}
}
