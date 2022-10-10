package types

// CompiledSource represents a source descriptor for a smart contract compilation, including AST and contained
// CompiledContract instances.
type CompiledSource struct {
	// Ast describes the abstract syntax tree artifact of a source file compilation, providing tokenization of the
	// source file components.
	Ast any

	// Contracts describes a mapping of contract names to contract definition structures which are contained within
	// the source.
	Contracts map[string]CompiledContract
}
