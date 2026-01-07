package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/crytic/medusa-geth/accounts/abi"
	"github.com/crytic/medusa-geth/common"
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

	// LibraryPlaceholders maps placeholder strings to library names (if known)
	// Format is map[placeholder]libraryName
	// When a contract has placeholders, these need to be resolved before deployment
	LibraryPlaceholders map[string]any
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

func (c *CompiledContract) DecodeLinkedInitBytecodeBytes() ([]byte, error) {
	return hex.DecodeString(strings.TrimPrefix(string(c.InitBytecode), "0x"))
}

func (c *CompiledContract) DecodeLinkedRuntimeBytecodeBytes() ([]byte, error) {
	return hex.DecodeString(strings.TrimPrefix(string(c.RuntimeBytecode), "0x"))
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

// LinkBytecodes replaces library placeholders in contract bytecode with actual deployed library addresses.
// This function is essential for contracts that depend on libraries, as it ensures the contract's bytecode
// correctly references the deployed library instances.
//
// The function performs the following steps:
// 1. Replaces library placeholders in both init and runtime bytecode
// 2. Decodes the hex-encoded bytecodes to bytes
// 3. Updates the contract's bytecode fields with the linked versions
//
// Parameters:
// - contractName: Name of the contract being linked (used for error reporting)
// - deployedLibraries: Map of library names to their deployed addresses
//
// If bytecode decoding fails, the function will panic with an appropriate error message.
func (c *CompiledContract) LinkBytecodes(contractName string, deployedLibraries map[string]common.Address) {
	if len(c.LibraryPlaceholders) == 0 {
		return
	}
	c.InitBytecode, c.RuntimeBytecode = getLinkedBytecodes(c.LibraryPlaceholders, c.InitBytecode, c.RuntimeBytecode, deployedLibraries)

	// Decode into hex string
	initBytecode, err := c.DecodeLinkedInitBytecodeBytes()
	if err != nil {
		panic(fmt.Errorf("unable to parse init bytecode for contract %s \n", contractName))
	}

	// Decode into a hex string
	runtimeBytecode, err := c.DecodeLinkedRuntimeBytecodeBytes()
	if err != nil {
		panic(fmt.Errorf("unable to parse runtime bytecode for contract %s \n", contractName))
	}
	c.InitBytecode = initBytecode
	c.RuntimeBytecode = runtimeBytecode

	// Clear the library placeholders map since they've been linked
	c.LibraryPlaceholders = make(map[string]any)
}

// ParseBytecodeForPlaceholders analyzes the given bytecode string to identify and extract
// all library placeholder patterns embedded within it.
//
// When a Solidity contract uses libraries, the compiler places placeholder patterns in the bytecode
// that later need to be replaced with actual library addresses. These placeholders follow the format
// "__$<placeholder>$__" or "__<identifier>__".
//
// This function:
// 1. Uses regex to find all library placeholder patterns in the bytecode
// 2. Extracts the unique placeholder identifiers by removing the surrounding delimiter characters
// 3. Creates a map where keys are the placeholder identifiers (without the "__" and "$" delimiters)
//
// Parameters:
// - bytecode: A string containing the contract bytecode to analyze
//
// Returns:
//   - A map where keys are the extracted placeholder identifiers and values are nil
//     (values will be populated later with library names when linking is performed)
//
// Note: If no library placeholders are found in the bytecode, returns an empty map.
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

// getLinkedBytecodes is a helper function that performs the actual replacement of library placeholders
// in contract bytecode with the addresses of deployed library instances.
//
// The Solidity compiler places special placeholder patterns in a contract's bytecode when it references
// external libraries. These patterns follow the format "__$<placeholder>$__" where <placeholder> is a
// unique identifier for each library dependency.
//
// Parameters:
// - libraryPlaceholders: Map of placeholder identifiers to library names
// - initBytecode: Contract initialization bytecode (used for deployment)
// - runtimeBytecode: Contract runtime bytecode (code that will be stored on-chain)
// - deployedLibraries: Map of library names to their deployed addresses
//
// Returns:
// - Modified initialization bytecode with library addresses inserted
// - Modified runtime bytecode with library addresses inserted
//
// Note: If a library is in the placeholder map but not found in deployedLibraries, its placeholder will
// remain in the bytecode. This function handles bytecode in hex string format and ensures proper formatting
// of library addresses.
func getLinkedBytecodes(libraryPlaceholders map[string]any, initBytecode []byte, runtimeBytecode []byte, deployedLibraries map[string]common.Address) ([]byte, []byte) {
	// Get the bytecode as a hex string
	initBytecodeHex := string(initBytecode)
	runtimeBytecodeHex := string(runtimeBytecode)

	// If it starts with 0x, remove it
	initBytecodeHex = strings.TrimPrefix(initBytecodeHex, "0x")
	runtimeBytecodeHex = strings.TrimPrefix(runtimeBytecodeHex, "0x")

	// Replace the placeholders with the deployed library addresses
	// For each library placeholder
	for placeholder, libNameAny := range libraryPlaceholders {
		libName, ok := libNameAny.(string)
		if !ok || libName == "" {
			continue
		}

		// Get the deployed library address
		libraryAddr, exists := deployedLibraries[libName]
		if !exists {
			continue
		}

		// The pattern in bytecode is "__$<placeholder>$__"
		placeholderPattern := fmt.Sprintf("__$%s$__", placeholder)

		// Get the address hex without "0x" prefix
		addrHex := libraryAddr.Hex()[2:]
		// Pad to 40 characters (20 bytes)
		for len(addrHex) < 40 {
			addrHex = "0" + addrHex
		}

		// Replace the placeholder with the address
		initBytecodeHex = strings.ReplaceAll(initBytecodeHex, placeholderPattern, addrHex)
		runtimeBytecodeHex = strings.ReplaceAll(runtimeBytecodeHex, placeholderPattern, addrHex)
	}
	// return the linked bytecode
	return []byte(initBytecodeHex), []byte(runtimeBytecodeHex)
}
