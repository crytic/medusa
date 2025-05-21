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
// based on their dependencies. This ensures libraries are deployed before contracts
// that depend on them.
//
// The function uses a modified Kahn's algorithm for topological sorting, with
// special handling for target contracts:
//  1. Nodes with no dependencies (in-degree = 0) from targetContracts are prioritized
//     in their original order
//  2. When multiple nodes become available for processing at the same time, nodes in
//     targetContracts are ordered according to their position in that list
//
// This way the deployment order respects both dependency constraints (libraries first)
// and preserves the desired order of target contracts when possible.
func GetDeploymentOrder(dependencies map[string][]string, targetContracts []string) ([]string, error) {
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
	// Create a map to track if a contract is in targetContracts for fast lookup
	targetContractsMap := make(map[string]int)
	for i, contract := range targetContracts {
		targetContractsMap[contract] = i
	}
	// First, add zero-degree nodes from targetContracts list in their original order
	for _, contract := range targetContracts {
		if degree, exists := inDegree[contract]; exists && degree == 0 {
			queue = append(queue, contract)
		}
	}
	// Then add remaining zero-degree nodes that aren't in targetContracts
	for node, degree := range inDegree {
		if degree == 0 {
			if _, inTargets := targetContractsMap[node]; !inTargets {
				queue = append(queue, node)
			}
		}
	}

	// Process nodes in topological order
	var result []string
	for len(queue) > 0 {
		// Remove a node from the queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)
		delete(inDegree, current)
		// For each node that depends on this one, decrease its in-degree
		for node, deps := range dependencies {
			for _, dep := range deps {
				if dep == current {
					inDegree[node]--
					if inDegree[node] == 0 {
						// When a node's degree becomes zero, check if it's in targetContracts
						// If it is, prioritize it based on its position in targetContracts
						if idx, inTargets := targetContractsMap[node]; inTargets {
							// Insert at the appropriate position based on target order
							inserted := false
							for i, queuedNode := range queue {
								if queuedIdx, queuedInTargets := targetContractsMap[queuedNode]; queuedInTargets && idx < queuedIdx {
									// Insert before this node (proper slice insertion)
									queue = append(queue, "")    // Add empty space
									copy(queue[i+1:], queue[i:]) // Shift elements right
									queue[i] = node              // Insert new element
									inserted = true
									break
								}
							}
							if !inserted {
								queue = append(queue, node)
							}
						} else {
							// For non-targeted contracts, just append them
							queue = append(queue, node)
						}
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
