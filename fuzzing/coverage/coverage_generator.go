package coverage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/compilation/types"
	"github.com/trailofbits/medusa/fuzzing/contracts"
)

// NOTE we probably should figure out a way to structure the contracts so that the coverage appears in the definitions themselves
// that said contract  definitions are probably not the best place as each definition represents "some contract deployment"
// and we could have multiple definitions of the same contract (I think)
type ExpandedContractData struct {
	ContractDefinition contracts.Contract

	coverageData codeCoverageData
}

// NOTE at the moment this is a clustefuck of a function
func GenerateCoverageReport(coverageMaps *CoverageMaps, contractDefinitions []contracts.Contract) {
	// initialize map
	coverageData := make(map[common.Hash]*ExpandedContractData)

	// The fuzzer generates coverage per address and codehash (e.g map[address] -> map[codeHash] -> coverage)
	// so for each codeHash we have to figure out to which contract definition it belongs to.
	// first we iterate over all the contract definitions and calculate the codeHashes then store them in a mapping
	for _, contractDefinition := range contractDefinitions {
		// extract contract metadata from runtime bytecode
		contractMetadata := types.ExtractContractMetadata(contractDefinition.CompiledContract().RuntimeBytecode)

		// if there is contract metadata then extract the codeHash
		if contractMetadata != nil {
			// NOTE: we have to convert from []byte to common.Hash perhaps ExtractBytecodeHash should return the type directly
			contractHash := common.BytesToHash(contractMetadata.ExtractBytecodeHash())

			// NOTE: perhaps this is not needed if the contract hashes are unique amongst definitions
			_, exists := coverageData[contractHash]
			if !exists {
				coverageData[contractHash] = &ExpandedContractData{contractDefinition, codeCoverageData{}}
			}

			// TODO: Figure out how to generate the bytecode hash when its not in the metadata
		} else {
			continue
		}
	}

	// aggregate all coverage data per contract definition
	for _, codeHashToCoverageMap := range coverageMaps.maps {
		// iterate over the hash to coverage mapping adding the new coverage data
		for codeHash, data := range codeHashToCoverageMap {
			_, err := coverageData[codeHash].coverageData.updateCodeCoverageData(data)
			if err != nil {
				// TODO: what do do if there is an error here ?
			}
		}
	}

	// At this point we have the aggregated coverage data for each one of the contract definitions
	// Now we need map the coverage data for each contract definition to its source file
	for _, contract := range coverageData {
		// We represent parsed bytecode as an array in which each element is an EVM intruction
		// for that EVM instruction we know at which offset in the bytecode it starts and at which it ends
		// therefore to generate coverage we can iterate over the parsed bytecode
		// get the parsed bytecode of the contract
		parsedRuntimeBytecode := contract.ContractDefinition.CompiledContract().ParsedRuntimeBytecode

		// get parsed source mapping
		parsedRuntimeSrceMap := contract.ContractDefinition.CompiledContract().ParsedRuntimeSrcMap

		// get the contract coverage data
		coverage := contract.coverageData.deployedBytecodeCoverageData

		// read and parse the source file
		// parsedFileData := splitSourceFileIntoLines(readSourceFile(contract.ContractDefinition.SourcePath()))

		// we need to iterate over the coverage data using the parsed bytecode as an index
		coverageIndex := 0

		// If there is coverage then update the parsed bytecode
		if len(coverage) != 0 {
			// iterate over the parsed bytecode marking instructions as covered (or not)
			for i, instruction := range parsedRuntimeBytecode {
				// if instruct was covered
				// NOTE: for now we assume that if the first byte of the instruction was covered so were the rest
				// which might not be the correct assumption
				if coverage[instruction.Start] == 1 {
					instruction.Covered = true
					parsedRuntimeSrceMap[i].Covered = true
				}

				// increment the index by the amount of bytes the instruction spawns
				coverageIndex = coverageIndex + instruction.End
			}
		}

		// generate coverage report

	}

}
