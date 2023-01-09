package types

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trailofbits/medusa/compilation/types"
)

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

// Placeholder represents the first 34 bytes of the keccak256 hash of the sourcePath concatenated, with a ":", to the
// name of the contract. This is used specifically to identify a library's placeholder.
func (c *Contract) Placeholder() string {
	// TODO: the 34 is a bit concerning
	placeholder := hex.EncodeToString(crypto.Keccak256([]byte(c.sourcePath + ":" + c.name)))[:34]
	return placeholder
}
