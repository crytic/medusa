package types

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"strings"
)

type CompiledContract struct {
	Abi abi.ABI
	InitBytecode string
	RuntimeBytecode string
	SrcMapsInit string
	SrcMapsRuntime string
}

type CompiledSource struct {
	Ast interface {}
	Contracts map[string]CompiledContract
}

type Compilation struct {
	Sources map[string]CompiledSource
}

func NewCompilation() *Compilation {
	// Create our compilation
	compilation := &Compilation{
		Sources: make(map[string]CompiledSource),
	}

	// Return the compilation.
	return compilation
}

func InterfaceToABI(i interface{}) (*abi.ABI, error) {
	// TODO: Refactor this ugly hack. Solidity 0.8.0 doesn't re-serialize ABI as a string, so go-ethereum simply
	//  ensures older Solidity deserializes too. It doesn't do it as abi.ABI type though, so we do that here
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