package types

// SourceArtifact represents a source descriptor for a smart contract compilation, including AST and contained
// CompiledContract instances.
type SourceArtifact struct {
	// Ast describes the abstract syntax tree artifact of a source file compilation, providing tokenization of the
	// source file components.
	Ast any

	// Contracts describes a mapping of contract names to contract definition structures which are contained within
	// the source.
	Contracts map[string]CompiledContract

	// SourceUnitId refers to the identifier of the source unit within the compilation.
	SourceUnitId int
}
