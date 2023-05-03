package types

import (
	"bytes"

	"github.com/fxamacker/cbor"
)

// ContractMetadata is an CBOR-encoded structure describing contract information which is embedded within smart contract
// bytecode by the Solidity compiler (unless explicitly directed not to).
// Reference: https://docs.soliditylang.org/en/v0.8.16/metadata.html
type ContractMetadata map[string]any

// metadataHashPrefixes defines patterns to use in search for CBOR-encoded contract metadata appended to the end of
// bytecode.
var metadataHashPrefixes = [][]byte{
	{0xa1, 0x65, 98, 122, 122, 114, 48, 0x58, 0x20},  // a1 65 "bzzr0" 0x58 0x20 (solc <= 0.5.8)
	{0xa2, 0x65, 98, 122, 122, 114, 48, 0x58, 0x20},  // a2 65 "bzzr0" 0x58 0x20 (solc >= 0.5.9)
	{0xa2, 0x65, 98, 122, 122, 114, 49, 0x58, 0x20},  // a2 65 "bzzr1" 0x58 0x20 (solc >= 0.5.11)
	{0xa2, 0x64, 0x69, 0x70, 0x66, 0x73, 0x58, 0x22}, // a2 64 "ipfs" 0x58 0x22 (solc >= 0.6.0)
}

// byteCodeHashMetadataKeys defines the keys in the CBOR-encoded ContractMetadata which contain bytecode hashes.
var byteCodeHashMetadataKeys = [...]string{
	"bzzr0",
	"bzzr1",
	"ipfs",
}

// ExtractContractMetadata extracts contract metadata from provided byte code and returns it. If contract metadata
// could not be extracted, nil is returned.
func ExtractContractMetadata(bytecode []byte) *ContractMetadata {
	// Try matching each metadata hash prefix in the file. Metadata is appended to the end of the file.
	for _, metadataHashPrefix := range metadataHashPrefixes {
		metadataOffset := bytes.LastIndex(bytecode, metadataHashPrefix[:])

		// If we found a match, decode the embedded metadata and return it.
		if metadataOffset != -1 {
			var metadata ContractMetadata
			err := cbor.Unmarshal(bytecode[metadataOffset:], &metadata)
			if err != nil {
				continue
			}
			return &metadata
		}
	}
	return nil
}

// RemoveContractMetadata takes bytecode and attempts to detect contract metadata within it, splitting it where the
// metadata is found.
// If contract metadata could be located, this method returns the bytecode solely (no contract metadata, and no
// constructor arguments, which tend to follow).
// Otherwise, this method returns the provided input as-is.
func RemoveContractMetadata(bytecode []byte) []byte {
	for _, metadataHashPrefix := range metadataHashPrefixes {
		metadataOffset := bytes.LastIndex(bytecode, metadataHashPrefix[:])

		if metadataOffset != -1 {
			return bytecode[:metadataOffset-1]
		}
	}
	return bytecode
}

// ExtractBytecodeHash extracts the bytecode hash from given contract metadata and returns the bytes representing the
// hash. If it could not be detected or extracted, nil is returned.
func (m ContractMetadata) ExtractBytecodeHash() []byte {
	// Try every known metadata key to see if we can resolve the bytecode hash
	for _, possibleMetadataKey := range byteCodeHashMetadataKeys {
		if bytecodeHashData, keyExists := m[possibleMetadataKey]; keyExists {
			// Try to cast it to a byte array and return it if we succeeded.
			if bytecodeHash, ok := bytecodeHashData.([]byte); ok {
				return bytecodeHash
			}
		}
	}
	return nil
}
