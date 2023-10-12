package main

import (
	"github.com/crytic/medusa/cmd"
	"net/http"
	_ "net/http/pprof"
	"os"
)

func main() {
	go func() {
		http.ListenAndServe("localhost:8080", nil)
	}()

	// Run our root CLI command, which contains all underlying command logic and will handle parsing/invocation.
	err := cmd.Execute()

	if err != nil {
		os.Exit(1)
	}
}
