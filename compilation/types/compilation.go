package types

import (
	"errors"
	"fmt"
	"os"
)

// Compilation represents the artifacts of a smart contract compilation.
type Compilation struct {
	// SourcePathToArtifact maps source file paths to their corresponding SourceArtifact.
	SourcePathToArtifact map[string]SourceArtifact

	// SourceIdToPath is a mapping of source unit IDs to source file paths.
	SourceIdToPath map[int]string

	// SourceCode is a lookup of a source file path from SourceList to source code. This is populated by
	// CacheSourceCode.
	SourceCode map[string][]byte
}

// NewCompilation returns a new, empty Compilation object.
func NewCompilation() *Compilation {
	// Create our compilation
	compilation := &Compilation{
		SourcePathToArtifact: make(map[string]SourceArtifact),
		SourceCode:           make(map[string][]byte),
		SourceIdToPath:       make(map[int]string),
	}

	// Return the compilation.
	return compilation
}

// CacheSourceCode caches source code for each CompiledSource in the compilation in the CompiledSource.SourceCode field.
// This method will attempt to populate each CompiledSource.SourceCode which has not yet been populated (is nil) before
// returning an error, if one occurs.
func (c *Compilation) CacheSourceCode() error {
	// Loop through each source file, try to read it, and collect errors in an aggregated string if we encounter any.
	var errStr string
	for sourcePath := range c.SourcePathToArtifact {
		if _, ok := c.SourceCode[sourcePath]; !ok {
			sourceCodeBytes, sourceReadErr := os.ReadFile(sourcePath)
			if sourceReadErr != nil {
				errStr += fmt.Sprintf("source file '%v' could not be cached due to error: '%v'\n", sourcePath, sourceReadErr)
			}
			c.SourceCode[sourcePath] = sourceCodeBytes
		}
	}

	// If we have an error message, return an error encapsulating it.
	if len(errStr) > 0 {
		return errors.New(errStr)
	}

	return nil
}
