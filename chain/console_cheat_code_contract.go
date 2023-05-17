package chain

import (
	"fmt"
	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"regexp"
	"strconv"
)

// getConsoleCheatCodeContract obtains a CheatCodeContract which implements the console.log cheatcodes.
// The Console precompile contract is returned if there are no errors.
func getConsoleCheatCodeContract(tracer *cheatCodeTracer) (*CheatCodeContract, error) {
	// Define our address for this precompile contract, then create a new precompile to add methods to.
	contractAddress := common.HexToAddress("0x000000000000000000636F6e736F6c652e6c6f67")
	contract := newCheatCodeContract(tracer, contractAddress, "Console")

	// Define all the ABI types needed for console.log functions
	typeUint256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	typeInt256, err := abi.NewType("int256", "", nil)
	if err != nil {
		return nil, err
	}
	typeString, err := abi.NewType("string", "", nil)
	if err != nil {
		return nil, err
	}
	typeBool, err := abi.NewType("bool", "", nil)
	if err != nil {
		return nil, err
	}
	typeAddress, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}
	typeBytes, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}

	// We will store all the fixed byte (e.g. byte1, byte2) in a mapping
	const numFixedByteTypes = 32
	fixedByteTypes := make(map[int]abi.Type, numFixedByteTypes)
	for i := 1; i <= numFixedByteTypes; i++ {
		byteString := "bytes" + strconv.FormatInt(int64(i), 10)
		fixedByteTypes[i], err = abi.NewType(byteString, "", nil)
		if err != nil {
			return nil, err
		}
	}

	// We have a few special log function signatures outside all the permutations of (string, int256, bool, address).
	// These include log(int256), log(bytes), log(bytesX), and log(string, int256). So, we will manually create these
	// signatures and then programmatically iterate through all the permutations.

	// log(int256): Log an int256
	intSig, err := contract.addEvent("Log", abi.Arguments{{Type: typeInt256}})
	if err != nil {
		return nil, err
	}
	contract.addMethod("log", abi.Arguments{{Type: typeInt256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Encode the data
			data, err := abi.Arguments{{Type: typeInt256}}.Pack(inputs...)
			if err != nil {
				return nil, cheatCodeRevertData([]byte("log: unable to pack the input data"))
			}

			// Create the log object
			log := types.Log{
				Address: contractAddress,
				Data:    data,
				Topics:  []common.Hash{intSig},
			}

			// Add it to the state DB
			tracer.evm.StateDB.AddLog(&log)
			return []any{}, nil
		},
	)

	// log(bytes): Log bytes
	bytesSig, err := contract.addEvent("Log", abi.Arguments{{Type: typeBytes}})
	if err != nil {
		return nil, err
	}
	contract.addMethod("log", abi.Arguments{{Type: typeBytes}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Encode the data
			data, err := abi.Arguments{{Type: typeBytes}}.Pack(inputs...)
			if err != nil {
				return nil, cheatCodeRevertData([]byte("log: unable to pack the input data"))
			}

			// Create the log object
			log := types.Log{
				Address: contractAddress,
				Data:    data,
				Topics:  []common.Hash{bytesSig},
			}

			// Add it to the state DB
			tracer.evm.StateDB.AddLog(&log)
			return []any{}, nil
		},
	)

	// Now, we will add the logBytes1, logBytes2, and so on in a loop
	for i := 1; i <= numFixedByteTypes; i++ {
		// Create local copy of abi argument
		fixedByteType := fixedByteTypes[i]

		// Create the event
		fixedByteSig, err := contract.addEvent("Log", abi.Arguments{{Type: fixedByteType}})
		if err != nil {
			return nil, err
		}

		// Add the method
		contract.addMethod("log", abi.Arguments{{Type: fixedByteType}}, abi.Arguments{},
			func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
				// Encode the data
				data, err := abi.Arguments{{Type: fixedByteType}}.Pack(inputs...)
				if err != nil {
					return nil, cheatCodeRevertData([]byte("log: unable to pack the input data"))
				}

				// Create the log object
				log := types.Log{
					Address: contractAddress,
					Data:    data,
					Topics:  []common.Hash{fixedByteSig},
				}

				// Add it to the state DB
				tracer.evm.StateDB.AddLog(&log)
				return []any{}, nil
			},
		)
	}

	// Add the string event here since we will need it for string formatting possibilities for the rest of the function
	stringEventSig, err := contract.addEvent("Log", abi.Arguments{{Type: typeString}})
	if err != nil {
		return nil, err
	}

	// log(string, int): Log string with an int where the string could be formatted
	stringIntSig, err := contract.addEvent("Log", abi.Arguments{{Type: typeString}, {Type: typeInt256}})
	if err != nil {
		return nil, err
	}
	contract.addMethod("log", abi.Arguments{{Type: typeString}, {Type: typeInt256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Create the log object
			log := types.Log{
				Address: contractAddress,
			}

			exp := regexp.MustCompile(`%`)
			// If the first input is a string and that input has a % in it, then we will string format the int
			stringInput, isString := inputs[0].(string)
			if isString && exp.MatchString(stringInput) {
				formattedString := fmt.Sprintf(inputs[0].(string), inputs[1:]...)
				abiEncodedString, err := abi.Arguments{{Type: typeString}}.Pack(formattedString)
				if err != nil {
					return nil, cheatCodeRevertData([]byte("log: unable to pack the formatted string"))
				}
				log.Data = abiEncodedString
				log.Topics = []common.Hash{stringEventSig}
			} else {
				// Otherwise, just pack the event data
				data, err := abi.Arguments{{Type: typeString}, {Type: typeInt256}}.Pack(inputs...)
				if err != nil {
					return nil, cheatCodeRevertData([]byte("log: unable to pack the provided input parameters"))
				}
				log.Data = data
				log.Topics = []common.Hash{stringIntSig}
			}

			// Add it to the state DB
			tracer.evm.StateDB.AddLog(&log)
			return []any{}, nil
		},
	)

	// These are the four parameter types that console.log() accepts
	choices := abi.Arguments{{Type: typeUint256}, {Type: typeString}, {Type: typeBool}, {Type: typeAddress}}

	// Create all possible permutations (with repetition) where the number of choices increases from 1...len(choices)
	permutations := make([]abi.Arguments, 0)
	for n := 1; n <= len(choices); n++ {
		nextSetOfPermutations := utils.PermutationsWithRepetition(choices, n)
		for _, permutation := range nextSetOfPermutations {
			permutations = append(permutations, permutation)
		}
	}

	// Iterate across each permutation to add their associated event and function handler
	for i := 0; i < len(permutations); i++ {
		// Add the event
		eventSig, err := contract.addEvent("Log", permutations[i])
		if err != nil {
			return nil, err
		}

		// Make a local copy of the current permutation
		permutation := permutations[i]

		// Create the function handler
		contract.addMethod("log", permutation, abi.Arguments{},
			func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
				// Create the log object
				log := types.Log{
					// We want the event address to be the caller of this cheatcode
					Address: contractAddress,
				}

				exp := regexp.MustCompile(`%`)
				// If the first input is a string and that input has a % in it, then there might be some string
				// formatting that needs to be taken care of
				stringInput, isString := inputs[0].(string)
				if isString && exp.MatchString(stringInput) {
					formattedString := fmt.Sprintf(inputs[0].(string), inputs[1:]...)
					abiEncodedString, err := abi.Arguments{{Type: typeString}}.Pack(formattedString)
					if err != nil {
						return nil, cheatCodeRevertData([]byte("log: unable to pack the formatted string"))
					}
					log.Data = abiEncodedString
					log.Topics = []common.Hash{stringEventSig}
				} else {
					// Otherwise, just pack the event data
					data, err := permutation.Pack(inputs...)
					if err != nil {
						return nil, cheatCodeRevertData([]byte("log: unable to pack the provided input parameters"))
					}
					log.Data = data
					log.Topics = []common.Hash{eventSig}
				}

				// Add it to the state DB
				tracer.evm.StateDB.AddLog(&log)
				return []any{}, nil
			},
		)
	}

	// Return our precompile contract information.
	return contract, nil
}
