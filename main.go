package main

import (
	"github.com/crytic/medusa/cmd"
	"github.com/crytic/medusa/utils"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

func main() {
	// Write heap profile to file every minute
	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			select {
			case <-ticker.C:
				filename := "heap.prof"
				f, _ := utils.CreateFile("pprof", filename)
				defer f.Close()
				runtime.GC()
				if err := pprof.WriteHeapProfile(f); err != nil {
					os.Exit(1)
				}
			}
		}
	}()

	// Have an HTTP endpoint for listening
	go func() {
		http.ListenAndServe("localhost:8080", nil)
	}()

	// Run our root CLI command, which contains all underlying command logic and will handle parsing/invocation.
	err := cmd.Execute()

	if err != nil {
		os.Exit(1)
	}
}
