package types

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/fxamacker/cbor"
	"github.com/trailofbits/medusa/compilation/types"
)

// DeployedContract describes a contract which is actively deployed on-chain at a given address.
type DeployedContract struct {
	// Address represents the Ethereum address where the deployed contract containing the method exists.
	Address common.Address

	// InitBytecode describes the bytecode used to deploy the contract.
	InitBytecode []byte

	// RuntimeBytecode describes the bytecode that exists after deployment of the contract.
	RuntimeBytecode []byte
}

// IsMatch returns a boolean indicating whether the deployed contract is a match with the provided compiled contract.
func (c *DeployedContract) IsMatch(contract *types.CompiledContract) bool {
	// TODO: Proper matching logic

	// Obtain the contract byte code as a byte slice.
	contractInitBytecode, err := contract.InitBytecodeBytes()
	if err != nil {
		return false
	}

	// Define patterns to search for CBOR-encoded contract metadata appended to the end of bytecode.
	metadataHashPrefixes := [][]byte{
		{0xa1, 0x65, 98, 122, 122, 114, 48, 0x58, 0x20},  // a1 65 "bzzr0" 0x58 0x20 (solc <= 0.5.8)
		{0xa2, 0x65, 98, 122, 122, 114, 48, 0x58, 0x20},  // a2 65 "bzzr0" 0x58 0x20 (solc >= 0.5.9)
		{0xa2, 0x65, 98, 122, 122, 114, 49, 0x58, 0x20},  // a2 65 "bzzr1" 0x58 0x20 (solc >= 0.5.11)
		{0xa2, 0x64, 0x69, 0x70, 0x66, 0x73, 0x58, 0x22}, // a2 64 "ipfs" 0x58 0x22 (solc >= 0.6.0)
	}

	// Define the keys in the CBOR-encoded contract metadata which contain bytecode hashes.
	byteCodeHashMetadataKeys := []string{
		"bzzr0",
		"bzzr1",
		"ipfs",
	}

	// Try matching each metadata hash prefix. Metadata is appended to the end of the file.
	// Reference: https://docs.soliditylang.org/en/v0.8.16/metadata.html
	for _, metadataHashPrefix := range metadataHashPrefixes {
		// Try to obtain an index of our search data in our deployed contract bytecode.
		deployedIndex := bytes.LastIndex(contractInitBytecode, metadataHashPrefix)
		if deployedIndex != -1 {
			// Try to obtain an index of our search data in our compiled contract bytecode.
			definitionIndex := bytes.LastIndex(c.InitBytecode, metadataHashPrefix)
			if definitionIndex != -1 {
				// Decode our deployed contract metadata
				var deployedMetadata map[string]any
				err = cbor.Unmarshal(contractInitBytecode[deployedIndex:], &deployedMetadata)
				if err != nil {
					continue
				}

				// Decode our compiled contract metadata
				var definitionMetadata map[string]any
				err = cbor.Unmarshal(c.InitBytecode[definitionIndex:], &definitionMetadata)
				if err != nil {
					continue
				}

				// Verify the embedded bytecode hashes
				for _, possibleMetadataKey := range byteCodeHashMetadataKeys {
					if deployedHash, deployedKeyExists := deployedMetadata[possibleMetadataKey]; deployedKeyExists {
						if definitionHash, definitionKeyExists := definitionMetadata[possibleMetadataKey]; definitionKeyExists {
							return bytes.Equal(deployedHash.([]byte), definitionHash.([]byte))
						}
					}
				}
			}
		}
	}

	// If the init byte code size is larger than what we initialized with, it is not a match.
	if len(c.InitBytecode) > len(contractInitBytecode) {
		return false
	}

	// As a last ditch effort, cut down the contract init bytecode to the size of the definition's to attempt to strip
	// away constructor arguments before performing a direct compare.
	contractInitBytecode = contractInitBytecode[:len(c.InitBytecode)]

	// If the byte code matches exactly, we treat this as a match.
	if bytes.Compare(c.InitBytecode, contractInitBytecode) == 0 {
		return true
	}

	return false
}
