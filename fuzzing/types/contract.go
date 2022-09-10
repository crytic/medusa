package types

import (
	"github.com/trailofbits/medusa/compilation/types"
)

// Contract describes a compiled smart contract.
type Contract struct {
	// name represents the name of the contract.
	name string

	// compiledContract describes the compiled contract data.
	compiledContract *types.CompiledContract
}

// NewContract returns a new Contract instance with the provided information.
func NewContract(name string, compiledContract *types.CompiledContract) *Contract {
	return &Contract{
		name:             name,
		compiledContract: compiledContract,
	}
}

// Name returns the name of the contract.
func (c *Contract) Name() string {
	return c.name
}

// CompiledContract returns the compiled contract information including source mappings, byte code, and ABI.
func (c *Contract) CompiledContract() *types.CompiledContract {
	return c.compiledContract
}
