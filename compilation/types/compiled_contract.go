package types

import (
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"strings"
)

// CompiledContract represents a single contract unit from a smart contract compilation.
type CompiledContract struct {
	// Abi describes a contract's application binary interface, a structure used to describe information needed
	// to interact with the contract such as constructor and function definitions with input/output variable
	// information, event declarations, and fallback and receive methods.
	Abi abi.ABI

	// InitBytecode describes the bytecode used to deploy a contract.
	InitBytecode string

	// RuntimeBytecode represents the rudimentary bytecode to be expected once the contract has been successfully
	// deployed. This may differ at runtime based on constructor arguments, immutables, linked libraries, etc.
	RuntimeBytecode string

	// SrcMapsInit describes the source mappings to associate source file and bytecode segments in InitBytecode.
	SrcMapsInit string

	// SrcMapsRuntime describes the source mappings to associate source file and bytecode segments in RuntimeBytecode.
	SrcMapsRuntime string
}

// InitBytecodeBytes returns the InitBytecode as a byte slice.
func (c *CompiledContract) InitBytecodeBytes() ([]byte, error) {
	return hex.DecodeString(strings.TrimPrefix(c.InitBytecode, "0x"))
}

// RuntimeBytecodeBytes returns the RuntimeBytecode as a byte slice.
func (c *CompiledContract) RuntimeBytecodeBytes() ([]byte, error) {
	return hex.DecodeString(strings.TrimPrefix(c.RuntimeBytecode, "0x"))
}

// ParseABIFromInterface parses a generic object into an abi.ABI and returns it, or an error if one occurs.
func ParseABIFromInterface(i interface{}) (*abi.ABI, error) {
	// TODO: Refactor this ugly hack. Solidity 0.8.0 doesn't re-serialize ABI as a string, so go-ethereum simply
	//  ensures older Solidity deserializes too. It doesn't do it as abi.ABI type though, so we do that here.
	b, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	result, err := abi.JSON(strings.NewReader(string(b)))
	if err != nil {
		return nil, err
	}
	return &result, nil
}
