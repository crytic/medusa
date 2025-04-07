package types

import (
	"encoding/hex"
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
