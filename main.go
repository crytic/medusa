package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/crytic/medusa/cmd"
	"github.com/crytic/medusa/cmd/exitcodes"
	"github.com/crytic/medusa/utils"
)

func main() {
	// Write heap profile to file every minute
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		i := 0
		for {
			select {
			case <-ticker.C:
				filename := "heap" + strconv.FormatInt(int64(i)%3, 10) + ".prof"
				f, _ := utils.CreateFile("pprof", filename)
				defer f.Close()
				runtime.GC()
				if err := pprof.WriteHeapProfile(f); err != nil {
					os.Exit(1)
				}
				i = i + 1
			}
		}
	}()

	// Have an HTTP endpoint for listening
	go func() {
		http.ListenAndServe("localhost:8080", nil)
	}()

	// Run our root CLI command, which contains all underlying command logic and will handle parsing/invocation.
	err := cmd.Execute()

	// Obtain the actual error and exit code from the error, if any.
	var exitCode int
	err, exitCode = exitcodes.GetInnerErrorAndExitCode(err)

	// If we have an error, print it.
	if err != nil && exitCode != exitcodes.ExitCodeHandledError {
		fmt.Println(err)
	}

	// If we have a non-success exit code, exit with it.
	if exitCode != exitcodes.ExitCodeSuccess {
		os.Exit(exitCode)
	}
}
