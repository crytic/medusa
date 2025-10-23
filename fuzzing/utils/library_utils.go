package utils

import (
	"container/heap"
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

// Priority queue storing contract names.
// Sorted by: predeploys first, then targetContracts, then the rest.
// We'll use heap.Push and heap.Pop on this queue.
type deploymentOrderPriorityQueue struct {
	items      []string
	priorities map[string]int
}

// In order to use heap.Push and heap.Pop on this queue, we need to implement heap.Interface.
// All these implementations should be fairly self explanatory...

func (pq deploymentOrderPriorityQueue) Len() int {
	return len(pq.items)
}

func (pq deploymentOrderPriorityQueue) Less(i, j int) bool {
	return pq.priorities[pq.items[i]] < pq.priorities[pq.items[j]]
}

func (pq deploymentOrderPriorityQueue) Swap(i, j int) {
	t := pq.items[i]
	pq.items[i] = pq.items[j]
	pq.items[j] = t
}

func (pq *deploymentOrderPriorityQueue) Push(x any) {
	pq.items = append(pq.items, x.(string))
}

func (pq *deploymentOrderPriorityQueue) Pop() any {
	lastIdx := len(pq.items) - 1
	item := pq.items[lastIdx]
	pq.items = pq.items[:lastIdx]
	return item
}

// GetDeploymentOrder returns a topologically sorted list of libraries/contracts
// based on their dependencies. This ensures libraries are deployed before contracts
// that depend on them.
//
// The function uses a modified Kahn's algorithm for topological sorting, with
// special handling for target contracts:
//  1. Nodes with no dependencies (in-degree = 0) from predeploys are prioritized
//     in their original order
//  2. Nodes with no dependencies (in-degree = 0) from targetContracts are prioritized
//     in their original order
//  3. When multiple nodes become available for processing at the same time, nodes in
//     predeploys are ordered according to their position in that list
//  4. When multiple nodes become available for processing at the same time, nodes in
//     targetContracts are ordered according to their position in that list
//
// This way the deployment order respects both dependency constraints (libraries first)
// and preserves the desired order of predeploys and target contracts when possible.
func GetDeploymentOrder(dependencies map[string][]string, predeploys []string, targetContracts []string) ([]string, error) {
	// Make the priority rankings for our priority queue. First predeploys (in the order they appear),
	// then targetContracts (in the order they appear), then dependencies (in the order that `range` gives us).
	priorities := make(map[string]int, len(dependencies))
	highestPriority := 0
	for _, c := range predeploys {
		if _, ok := priorities[c]; ok {
			continue
		}
		highestPriority++
		priorities[c] = highestPriority
	}
	for _, c := range targetContracts {
		if _, ok := priorities[c]; ok {
			continue
		}
		highestPriority++
		priorities[c] = highestPriority
	}
	for c := range dependencies {
		if _, ok := priorities[c]; ok {
			continue
		}
		highestPriority++
		priorities[c] = highestPriority
	}

	// Priority queue holding nodes with no dependencies (in-degree = 0)
	queue := &deploymentOrderPriorityQueue{
		items:      make([]string, 0, len(dependencies)),
		priorities: priorities,
	}

	// Map holding the in-degree for each node (number of dependencies)
	inDegree := make(map[string]int, len(dependencies))
	// Map that tracks, for each node, the nodes that depend on it.
	// This is the transpose graph of `dependencies`.
	dependenciesFrom := make(map[string][]string, len(dependencies))
	// Calculate inDegree, dependenciesFrom, and populate queue
	for node, deps := range dependencies {
		// Each node's in-degree is initially the number of its dependencies
		inDegree[node] = len(deps)
		// If a node has no dependencies, add to queue
		if len(deps) == 0 {
			heap.Push(queue, node)
		}
		// Add dependenciesFrom entries
		for _, dep := range deps {
			dependenciesFrom[dep] = append(dependenciesFrom[dep], node)
		}
	}

	// Process nodes in topological order
	result := make([]string, 0, len(dependencies))
	for queue.Len() > 0 {
		// Remove a node from the queue
		current := heap.Pop(queue).(string)
		result = append(result, current)
		// For each node that depends on this one, decrease its in-degree
		for _, node := range dependenciesFrom[current] {
			inDegree[node]--
			if inDegree[node] == 0 {
				// When a node's degree becomes zero, add it to the priority queue
				heap.Push(queue, node)
			}
		}
	}

	// Check if we have a valid topological ordering
	if len(result) != len(dependencies) {
		return result, fmt.Errorf("circular dependency detected in library dependencies")
	}

	return result, nil
}
