package types

import (
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
	InitBytecode []byte

	// RuntimeBytecode represents the rudimentary bytecode to be expected once the contract has been successfully
	// deployed. This may differ at runtime based on constructor arguments, immutables, linked libraries, etc.
	RuntimeBytecode []byte

	// SrcMapsInit describes the source mappings to associate source file and bytecode segments in InitBytecode.
	SrcMapsInit string

	// SrcMapsRuntime describes the source mappings to associate source file and bytecode segments in RuntimeBytecode.
	SrcMapsRuntime string
}

// ParseABIFromInterface parses a generic object into an abi.ABI and returns it, or an error if one occurs.
func ParseABIFromInterface(i any) (*abi.ABI, error) {
	var (
		result abi.ABI
		err    error
	)

	// If it's a string, just parse it. Otherwise, we assume it's an interface and serialize it into a string.
	if s, ok := i.(string); ok {
		result, err = abi.JSON(strings.NewReader(s))
		if err != nil {
			return nil, err
		}
	} else {
		var b []byte
		b, err = json.Marshal(i)
		if err != nil {
			return nil, err
		}
		result, err = abi.JSON(strings.NewReader(string(b)))
		if err != nil {
			return nil, err
		}
	}
	return &result, nil
}
