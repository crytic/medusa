package types

import (
	"encoding/hex"
	"fmt"
	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/crypto"
	"path/filepath"
)

// GenerateLibraryPlaceholder creates a library placeholder based on the keccak256 hash
// of the fully qualified library name according to Solidity's algorithm
func GenerateLibraryPlaceholder(fullyQualifiedName string) string {
	// Calculate keccak256 hash of the library name
	hash := crypto.Keccak256Hash([]byte(fullyQualifiedName))

	// Take the first 34 characters of the hash (17 bytes)
	hashStr := hex.EncodeToString(hash.Bytes())
	placeholderHash := hashStr[:34]

	// Format according to Solidity's placeholder format: __$<hash>$__
	return placeholderHash
}

// MapPlaceholdersToLibraries generates a mapping of placeholders to library names
// by computing the keccak256 hash of each library's fully qualified name
func MapPlaceholdersToLibraries(placeholderToLibrary map[string]any, availableLibraries map[string]string) {

	// For each library, calculate its expected placeholder and check if it's in the bytecode
	for fullName, shortName := range availableLibraries {
		placeholder := GenerateLibraryPlaceholder(fullName)

		// Check if this placeholder exists in the bytecode
		for p := range placeholderToLibrary {
			if p == placeholder {
				placeholderToLibrary[placeholder] = shortName
				break
			}
		}
	}
}

// GetAvailableLibraries builds a map of full library names to short names
// from the compilation results
func GetAvailableLibraries(compilations []Compilation) (map[string]string, string) {
	libraryMap := make(map[string]string)

	// Process each compilation unit
	for _, compilation := range compilations {
		// Go through each source file
		for sourcePath, sourceArtifact := range compilation.SourcePathToArtifact {
			// Check each contract in the source
			for contractName, contract := range sourceArtifact.Contracts {
				// Check if this is a library
				if contract.Kind == ContractKindLibrary {
					// Full name is "sourcePath:contractName"
					libPath := filepath.Join(filepath.Base(filepath.Dir(sourcePath)), filepath.Base(sourcePath))
					fullName := libPath + ":" + contractName
					// Short name is just the contract name
					shortName := contractName
					libraryMap[fullName] = shortName
				}
			}
		}
	}
	return libraryMap, ""
}

// ReplacePlaceholdersInBytecode replaces library placeholders in bytecode with actual library addresses
func ReplacePlaceholdersInBytecode(bytecode []byte, libraryPlaceholders map[string]any, deployedLibraries map[string]common.Address) []byte {
	// Clone the bytecode to avoid modifying the original
	result := make([]byte, len(bytecode))
	copy(result, bytecode)

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

		// Find the placeholder pattern in the bytecode
		// The full pattern in bytecode is: __$<placeholder>$__
		fullPattern := fmt.Sprintf("__%s__", placeholder)

		// Replace all occurrences in the bytecode
		// Since we're working with bytes, we need to do this manually
		for i := 0; i <= len(result)-len(fullPattern); i++ {
			match := true
			for j := 0; j < len(fullPattern); j++ {
				if i+j >= len(result) || result[i+j] != fullPattern[j] {
					match = false
					break
				}
			}

			if match {
				// Replace the placeholder with the address (padded)
				// The address needs to be exactly the same length as the placeholder
				addrBytes := libraryAddr.Bytes()
				for j := 0; j < len(addrBytes) && j < len(fullPattern); j++ {
					result[i+j] = addrBytes[j]
				}
				// Pad remaining bytes with zeros if needed
				for j := len(addrBytes); j < len(fullPattern); j++ {
					result[i+j] = 0
				}
			}
		}
	}

	return result
}

// GetDeploymentOrder returns a topologically sorted list of libraries/contracts
// based on their dependencies (libraries that other libraries depend on come first)
func GetDeploymentOrder(contractDependencies map[string][]any) ([]string, error) {
	// Convert to a map of string -> []string for easier processing
	dependencies := make(map[string][]string)
	for contract, deps := range contractDependencies {
		dependencies[contract] = make([]string, 0)
		for _, depAny := range deps {
			if depStr, ok := depAny.(string); ok && depStr != "" {
				dependencies[contract] = append(dependencies[contract], depStr)
			}
		}
	}

	// Calculate in-degree for each node (number of dependencies)
	inDegree := make(map[string]int)

	// Count incoming edges (dependencies)
	for node, deps := range dependencies {
		// Each node's in-degree is initially the number of its dependencies
		inDegree[node] = len(deps)

		// Make sure all dependencies exist in the inDegree map
		for _, dep := range deps {
			if _, exists := inDegree[dep]; !exists {
				inDegree[dep] = 0
			}
		}
	}

	// Find nodes with no dependencies (in-degree = 0)
	var queue []string
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	// Process nodes in topological order
	var result []string
	for len(queue) > 0 {
		// Remove a node from the queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// For each node that depends on this one, decrease its in-degree
		for node, deps := range dependencies {
			for _, dep := range deps {
				if dep == current {
					inDegree[node]--
					if inDegree[node] == 0 {
						queue = append(queue, node)
					}
				}
			}
		}
	}

	// Check if we have a valid topological ordering
	if len(result) != len(dependencies) {
		return result, fmt.Errorf("circular dependency detected in library dependencies")
	}

	return result, nil
}
