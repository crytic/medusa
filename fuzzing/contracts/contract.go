package contracts

import (
	"github.com/crytic/medusa/compilation/types"
)

// Contracts describes an array of contracts
type Contracts []*Contract

// MatchBytecode takes init and/or runtime bytecode and attempts to match it to a contract definition in the
// current list of contracts. It returns the contract definition if found. Otherwise, it returns nil.
func (c Contracts) MatchBytecode(initBytecode []byte, runtimeBytecode []byte) *Contract {
	// Loop through all our contract definitions to find a match.
	for i := 0; i < len(c); i++ {
		// If we have a match, register the deployed contract.
		if c[i].CompiledContract().IsMatch(initBytecode, runtimeBytecode) {
			return c[i]
		}
	}

	// If we found no definition, return nil.
	return nil
}

// Contract describes a compiled smart contract.
type Contract struct {
	// name represents the name of the contract.
	name string

	// sourcePath represents the key used to index the source file in the compilation it was derived from.
	sourcePath string

	// compiledContract describes the compiled contract data.
	compiledContract *types.CompiledContract
}

// NewContract returns a new Contract instance with the provided information.
func NewContract(name string, sourcePath string, compiledContract *types.CompiledContract) *Contract {
	return &Contract{
		name:             name,
		sourcePath:       sourcePath,
		compiledContract: compiledContract,
	}
}

// Name returns the name of the contract.
func (c *Contract) Name() string {
	return c.name
}

// SourcePath returns the path of the source file containing the contract.
func (c *Contract) SourcePath() string {
	return c.sourcePath
}

// CompiledContract returns the compiled contract information including source mappings, byte code, and ABI.
func (c *Contract) CompiledContract() *types.CompiledContract {
	return c.compiledContract
}
