package types

// Compilation represents the artifacts of a smart contract compilation.
type Compilation struct {
	// Sources describes the CompiledSource objects provided in a compilation, housing information regarding source
	// files, mappings, ASTs, and contracts.
	Sources map[string]CompiledSource
}

// NewCompilation returns a new, empty Compilation object.
func NewCompilation() *Compilation {
	// Create our compilation
	compilation := &Compilation{
		Sources: make(map[string]CompiledSource),
	}

	// Return the compilation.
	return compilation
}
