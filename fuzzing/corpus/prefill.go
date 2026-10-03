package corpus

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/crytic/medusa-geth/accounts/abi"
	"github.com/crytic/medusa-geth/common"
	compilationTypes "github.com/crytic/medusa/compilation/types"
	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/logging"
)

// PrefillCorpusFromTests attempts to extract call sequences from existing Foundry test functions
// and add them to the corpus. This helps seed the fuzzer with known-good transaction sequences.
// Returns an error if extraction fails critically, or nil if extraction succeeds or fails gracefully.
func (c *Corpus) PrefillCorpusFromTests(contractDefinitions contracts.Contracts, compilations []compilationTypes.Compilation) error {
	logger := c.logger.NewSubLogger("module", "corpus-prefill")
	logger.Info("Attempting to prefill corpus from Foundry tests")

	// Extract test sequences from AST
	sequences, err := extractTestSequencesFromAST(compilations, contractDefinitions, logger)
	if err != nil {
		logger.Warn("Failed to extract test sequences: " + err.Error())
		return nil // Non-fatal - continue without prefilling
	}

	if len(sequences) == 0 {
		logger.Info("No test sequences found to prefill corpus")
		return nil
	}

	// Add extracted sequences to the corpus
	addedCount := 0
	for _, sequence := range sequences {
		if len(sequence) > 0 {
			// Add with a default weight of 1
			err = c.addCallSequence(c.callSequenceFiles, sequence, true, big.NewInt(1), false)
			if err != nil {
				logger.Warn(fmt.Sprintf("Failed to add prefilled sequence: %v", err))
				continue
			}
			addedCount++
		}
	}

	logger.Info(fmt.Sprintf("Successfully prefilled corpus with %d test sequences", addedCount))

	// Flush all sequences to disk
	if addedCount > 0 {
		return c.Flush()
	}

	return nil
}

// extractTestSequencesFromAST analyzes test contracts using AST and extracts call sequences
func extractTestSequencesFromAST(compilations []compilationTypes.Compilation, contractDefinitions contracts.Contracts, logger *logging.Logger) ([]calls.CallSequence, error) {
	sequences := make([]calls.CallSequence, 0)

	// Iterate through all compilations and source artifacts
	for i := range compilations {
		compilation := &compilations[i]
		for _, sourceArtifact := range compilation.SourcePathToArtifact {
			// Parse AST if available
			if sourceArtifact.Ast == nil {
				continue
			}

			astData, err := json.Marshal(sourceArtifact.Ast)
			if err != nil {
				continue
			}

			var ast compilationTypes.AST
			if err := json.Unmarshal(astData, &ast); err != nil {
				logger.Debug(fmt.Sprintf("Failed to parse AST: %v", err))
				continue
			}

			// Find test contracts
			for _, node := range ast.Nodes {
				contractDef, ok := node.(compilationTypes.ContractDefinition)
				if !ok {
					continue
				}

				// Check if this is a test contract
				if !compilationTypes.IsTestContract(contractDef) {
					continue
				}

				// Extract calls from test functions
				for _, funcNode := range contractDef.Nodes {
					funcDef, ok := funcNode.(compilationTypes.FunctionDefinition)
					if !ok {
						continue
					}

					// Check if this is a test function
					if !compilationTypes.IsTestFunction(funcDef) {
						continue
					}

					// Extract calls from the function
					extractedCalls, err := compilationTypes.ExtractFunctionCalls(funcDef)
					if err != nil {
						logger.Debug(fmt.Sprintf("Failed to extract calls from %s.%s: %v", contractDef.CanonicalName, funcDef.Name, err))
						continue
					}

					if len(extractedCalls) == 0 {
						continue
					}

					// Convert to CallSequence
					sequence := convertExtractedCallsToSequence(extractedCalls, contractDefinitions, logger)
					if len(sequence) > 0 {
						sequences = append(sequences, sequence)
					}
				}
			}
		}
	}

	return sequences, nil
}

// convertExtractedCallsToSequence converts extracted calls to Medusa's CallSequence format
func convertExtractedCallsToSequence(extractedCalls []compilationTypes.ExtractedCall, contractDefinitions contracts.Contracts, logger *logging.Logger) calls.CallSequence {
	sequence := make(calls.CallSequence, 0, len(extractedCalls))

	for _, call := range extractedCalls {
		// Find the contract definition
		// Note: We do case-insensitive matching because variable names (e.g., "counter")
		// often differ in case from contract names (e.g., "Counter")
		var targetContract *contracts.Contract
		for _, contract := range contractDefinitions {
			if strings.EqualFold(contract.Name(), call.ContractName) {
				targetContract = contract
				break
			}
		}
		if targetContract == nil {
			logger.Debug(fmt.Sprintf("Contract '%s' not found in definitions", call.ContractName))
			continue
		}

		// Find the method in the ABI by name
		method := findMethodInABIByName(targetContract.CompiledContract().Abi, call.FunctionName)
		if method == nil {
			logger.Debug(fmt.Sprintf("Method '%s' not found in contract '%s'", call.FunctionName, call.ContractName))
			continue
		}

		// Create a call message with zero/default arguments
		// The fuzzer will mutate these values
		args := make([]any, len(method.Inputs))
		for i, input := range method.Inputs {
			args[i] = getZeroValueForType(input.Type)
		}

		// Use default addresses
		fromAddr := common.HexToAddress("0x10000")
		toAddr := common.HexToAddress("0x0") // Will be resolved at runtime

		// Create the call message
		callMsg := &calls.CallMessage{
			From:     fromAddr,
			To:       &toAddr,
			Nonce:    0,
			Value:    big.NewInt(0),
			GasLimit: 12000000,
			GasPrice: big.NewInt(1),
			Data:     nil, // Will be set via DataAbiValues
			DataAbiValues: &calls.CallMessageDataAbiValues{
				Method:      method,
				InputValues: args,
			},
		}

		// Create the call sequence element
		element := calls.NewCallSequenceElement(targetContract, callMsg, 0, 0)
		sequence = append(sequence, element)
	}

	return sequence
}

// findMethodInABIByName finds a method in an ABI by its name
func findMethodInABIByName(contractAbi abi.ABI, name string) *abi.Method {
	for _, method := range contractAbi.Methods {
		if method.Name == name {
			return &method
		}
	}
	return nil
}

// getZeroValueForType returns a zero value for a given ABI type
func getZeroValueForType(t abi.Type) any {
	switch t.T {
	case abi.IntTy, abi.UintTy:
		return big.NewInt(0)
	case abi.BoolTy:
		return false
	case abi.StringTy:
		return ""
	case abi.AddressTy:
		return common.Address{}
	case abi.BytesTy, abi.FixedBytesTy:
		return []byte{}
	case abi.SliceTy, abi.ArrayTy:
		return []any{}
	default:
		return nil
	}
}
