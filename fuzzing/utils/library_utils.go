package utils

import (
	"encoding/hex"
	"fmt"
	"github.com/crytic/medusa-geth/crypto"
	"github.com/crytic/medusa/compilation/types"
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

// ResolvePlaceholderLibraryReferences resolves library placeholder references to their actual library names
// by identifying placeholders and mapping them to the corresponding library using the fully qualified library name's hash
func ResolvePlaceholderLibraryReferences(placeholderToLibrary map[string]any, availableLibraries map[string]string) {

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

// BuildLibraryNameMapping builds a map of fully qualified library names to short names
// by scanning the compilation artifacts for contracts of type "library".
//
// Fully qualified names (like "path/to/File.sol:LibraryName") are used for placeholder generation,
// while short names ("LibraryName") are used for linking references in contract bytecode.
func BuildLibraryNameMapping(compilations []types.Compilation) map[string]string {
	libraryMap := make(map[string]string)

	// Process each compilation unit
	for _, compilation := range compilations {
		// Go through each source file
		for _, sourceArtifact := range compilation.SourcePathToArtifact {
			// Get absolute path
			libPath := ""
			if astMap, ok := sourceArtifact.Ast.(map[string]any); ok {
				for astKey := range astMap {
					if astKey == "absolutePath" {
						libPath, _ = astMap[astKey].(string)
					}
				}
			}
			// Throw error if for any reason libPath becomes empty
			if libPath == "" {
				panic("libPath is empty, could not determine the library path")
			}
			// Check each contract in the source
			for contractName, contract := range sourceArtifact.Contracts {
				// Check if this is a library
				if contract.Kind == types.ContractKindLibrary {
					fullName := libPath + ":" + contractName
					// Short name is just the contract name
					shortName := contractName
					libraryMap[fullName] = shortName
				}
			}
		}
	}
	return libraryMap
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
