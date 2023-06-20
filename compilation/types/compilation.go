package types

import (
	"errors"
	"fmt"
	"golang.org/x/exp/slices"
	"os"
)

// Compilation represents the artifacts of a smart contract compilation.
type Compilation struct {
	// Sources describes the CompiledSource objects provided in a compilation, housing information regarding source
	// files, mappings, ASTs, and contracts.
	Sources map[string]CompiledSource

	// SourceList describes the CompiledSource keys in Sources, in order. The file identifier used for a SourceMap
	// corresponds to an index in this list.
	SourceList []string

	// SourceCode is a lookup of a source file path from SourceList to source code. This is populated by
	// CacheSourceCode.
	SourceCode map[string][]byte
}

// NewCompilation returns a new, empty Compilation object.
func NewCompilation() *Compilation {
	// Create our compilation
	compilation := &Compilation{
		Sources:    make(map[string]CompiledSource),
		SourceList: make([]string, 0),
		SourceCode: make(map[string][]byte),
	}

	// Return the compilation.
	return compilation
}

// GetSourceFileId obtains the file identifier for a given source file path. This simply checks the index of the
// source file path in SourceList.
// Returns the identifier of the source file, or -1 if it could not be found.
func (c *Compilation) GetSourceFileId(sourcePath string) int {
	return slices.Index(c.SourceList, sourcePath)
}

// CacheSourceCode caches source code for each CompiledSource in the compilation in the CompiledSource.SourceCode field.
// This method will attempt to populate each CompiledSource.SourceCode which has not yet been populated (is nil) before
// returning an error, if one occurs.
func (c *Compilation) CacheSourceCode() error {
	// Loop through each source file, try to read it, and collect errors in an aggregated string if we encounter any.
	var errStr string
	for sourcePath := range c.Sources {
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
