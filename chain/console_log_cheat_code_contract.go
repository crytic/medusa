package chain

import (
	"strconv"

	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// ConsoleLogContractAddress is the address for the console.log precompile contract
var ConsoleLogContractAddress = common.HexToAddress("0x000000000000000000636F6e736F6c652e6c6f67")

// getConsoleLogCheatCodeContract obtains a CheatCodeContract which implements the console.log functions.
// Returns the precompiled contract, or an error if there is one.
func getConsoleLogCheatCodeContract(tracer *cheatCodeTracer) (*CheatCodeContract, error) {
	// Create a new precompile to add methods to.
	contract := newCheatCodeContract(tracer, ConsoleLogContractAddress, "Console")

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

	// We have a few special log function signatures outside all the permutations of (string, uint256, bool, address).
	// These include log(int256), log(bytes), log(bytesX), and log(string, uint256). So, we will manually create these
	// signatures and then programmatically iterate through all the permutations.

	// Note that none of the functions actually do anything - they just have to be callable so that the execution
	// traces can show the arguments that the user wants to log!

	// log(int256): Log an int256
	contract.addMethod("log", abi.Arguments{{Type: typeInt256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			return nil, nil
		},
	)

	// log(bytes): Log bytes
	contract.addMethod("log", abi.Arguments{{Type: typeBytes}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			return nil, nil
		},
	)

	// Now, we will add the logBytes1, logBytes2, and so on in a loop
	for i := 1; i <= numFixedByteTypes; i++ {
		// Create local copy of abi argument
		fixedByteType := fixedByteTypes[i]

		// Add the method
		contract.addMethod("log", abi.Arguments{{Type: fixedByteType}}, abi.Arguments{},
			func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
				return nil, nil
			},
		)
	}

	// log(string, int256): Log string with an int where the string could be formatted
	contract.addMethod("log", abi.Arguments{{Type: typeString}, {Type: typeInt256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			return nil, nil
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
		// Make a local copy of the current permutation
		permutation := permutations[i]

		// Create the function handler
		contract.addMethod("log", permutation, abi.Arguments{},
			func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
				return nil, nil
			},
		)
	}

	// Return our precompile contract information.
	return contract, nil
}
