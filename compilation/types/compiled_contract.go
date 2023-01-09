package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"regexp"
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

	// PlaceholderSet describes the set of library placeholders identified in the contract that need to be replaced with
	// library addresses at deploy-time
	PlaceholderSet map[string]any
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

// ParseBytecodeForPlaceholders will parse bytecode for all library placeholders and return them as a set. These placeholders will be
// replaced with the addresses of the deployed libraries at deploy-time.
func ParseBytecodeForPlaceholders(bytecode string) map[string]any {
	// Identify all library placeholder substrings
	exp := regexp.MustCompile(`__(\$[0-9a-zA-Z]*\$|\w*)__`)
	substrings := exp.FindAllString(bytecode, -1)

	substringSet := make(map[string]any, 0)

	// If we have no matches, then no linking is required, so return an empty set
	if substrings == nil {
		return substringSet
	}

	// Identify all unique library substrings
	for _, substring := range substrings {
		// Strip all `_` and `$` from the substring
		substring = strings.ReplaceAll(strings.ReplaceAll(substring, "_", ""), "$", "")

		// Only add it to the set if it is not already in it
		if _, exists := substringSet[substring]; !exists {
			substringSet[substring] = nil
		}
	}

	return substringSet
}

// InitBytecodeBytes returns the InitBytecode as a byte slice. Note that this function is guaranteed to return an error
// if the CompiledContract needs library linking.
func (c *CompiledContract) InitBytecodeBytes() ([]byte, error) {
	return hex.DecodeString(strings.TrimPrefix(c.InitBytecode, "0x"))
}

// RuntimeBytecodeBytes returns the RuntimeBytecode as a byte slice. Note that this function is guaranteed to return an error
// if the CompiledContract needs library linking.
func (c *CompiledContract) RuntimeBytecodeBytes() ([]byte, error) {
	return hex.DecodeString(strings.TrimPrefix(c.RuntimeBytecode, "0x"))
}

// IsLibrary checks to see whether a CompiledContract is a library or not. If the first LibraryIndicatorLength characters
// in the runtime bytecode of the CompiledContract is LibraryIndicator, then the contract is a library.
// See https://docs.soliditylang.org/en/v0.8.17/contracts.html#call-protection-for-libraries
func (c *CompiledContract) IsLibrary() bool {
	return c.RuntimeBytecode[:LibraryIndicatorLength] == LibraryIndicator
}

// LinkInitAndRuntimeBytecode will link all libraries to the init and runtime bytecode of a contract.
func (c *CompiledContract) LinkInitAndRuntimeBytecode(placeholderToLibraryAddress map[string]common.Address) error {
	// Remove the `_` and `$` special characters
	initBytecode := strings.ReplaceAll(strings.ReplaceAll(c.InitBytecode, "_", ""), "$", "")
	runtimeBytecode := strings.ReplaceAll(strings.ReplaceAll(c.RuntimeBytecode, "_", ""), "$", "")

	// Replace each placeholder with its associated address or throw an error if we cannot find the associated placeholder
	for placeholder := range c.PlaceholderSet {
		if libraryAddress, found := placeholderToLibraryAddress[placeholder]; !found {
			return fmt.Errorf("unable to find the following placeholder %v\n in this init bytecode: %v\n", placeholder, initBytecode)
		} else {
			libraryAddressString := strings.TrimPrefix(libraryAddress.String(), "0x")
			initBytecode = strings.ReplaceAll(initBytecode, placeholder, libraryAddressString)
			runtimeBytecode = strings.ReplaceAll(runtimeBytecode, placeholder, libraryAddressString)
		}
	}

	// Update the stored values
	c.InitBytecode = initBytecode
	c.RuntimeBytecode = runtimeBytecode
	return nil
}
