package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"golang.org/x/exp/slices"
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

	// Kind describes the kind of contract, i.e. contract, library, interface.
	Kind ContractKind
}

// IsMatch returns a boolean indicating whether provided contract bytecode is a match to this compiled contract
// definition.
func (c *CompiledContract) IsMatch(initBytecode []byte, runtimeBytecode []byte) bool {
	// Check if we can compare init and runtime bytecode
	canCompareInit := len(initBytecode) > 0 && len(c.InitBytecode) > 0
	canCompareRuntime := len(runtimeBytecode) > 0 && len(c.RuntimeBytecode) > 0
	// First try matching runtime bytecode contract metadata.
	if canCompareRuntime {
		// First we try to match contracts with contract metadata embedded within the smart contract.
		// Note: We use runtime bytecode for this because init byte code can have matching metadata hashes for different
		// contracts.
		deploymentMetadata := ExtractContractMetadata(runtimeBytecode)
		definitionMetadata := ExtractContractMetadata(c.RuntimeBytecode)
		if deploymentMetadata != nil && definitionMetadata != nil {
			deploymentBytecodeHash := deploymentMetadata.ExtractBytecodeHash()
			definitionBytecodeHash := definitionMetadata.ExtractBytecodeHash()
			if deploymentBytecodeHash != nil && definitionBytecodeHash != nil {
				return bytes.Equal(deploymentBytecodeHash, definitionBytecodeHash)
			}
		}
	}

	// Since we could not match with runtime bytecode's metadata hashes, we try to match based on init code. To do this,
	// we anticipate our init bytecode might contain appended arguments, so we'll be slicing it down to size and trying
	// to match as a last ditch effort.
	if canCompareInit {
		// If the init byte code size is larger than what we initialized with, it is not a match.
		if len(c.InitBytecode) > len(initBytecode) {
			return false
		}

		// Cut down the contract init bytecode to the size of the definition's to attempt to strip away constructor
		// arguments before performing a direct compare.
		cutDeployedInitBytecode := initBytecode[:len(c.InitBytecode)]

		// If the byte code matches exactly, we treat this as a match.
		if bytes.Equal(cutDeployedInitBytecode, c.InitBytecode) {
			return true
		}
	}

	// As a final fallback, try to compare the whole runtime byte code (least likely to work, given the deployment
	// process, e.g. smart contract constructor, will change the runtime code in most cases).
	if canCompareRuntime {
		// If the byte code matches exactly, we treat this as a match.
		if bytes.Equal(runtimeBytecode, c.RuntimeBytecode) {
			return true
		}
	}

	// Otherwise return our failed match status.
	return false
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

// GetDeploymentMessageData is a helper method used create contract deployment message data for the given contract.
// This data can be set in transaction/message structs "data" field to indicate the packed init bytecode and constructor
// argument data to use.
func (c *CompiledContract) GetDeploymentMessageData(args []any) ([]byte, error) {
	// ABI encode constructor arguments and append them to the end of the bytecode
	initBytecodeWithArgs := slices.Clone(c.InitBytecode)
	if len(c.Abi.Constructor.Inputs) > 0 {
		data, err := c.Abi.Pack("", args...)
		if err != nil {
			return nil, fmt.Errorf("could not encode constructor arguments due to error: %v", err)
		}
		initBytecodeWithArgs = append(initBytecodeWithArgs, data...)
	}
	return initBytecodeWithArgs, nil
}
