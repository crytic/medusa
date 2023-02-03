package chain

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// getCheatCodeProviders obtains a cheatCodeTracer (used to power cheat code analysis) and associated cheatCodeContract
// objects linked to the tracer (providing on-chain callable methods as an entry point). These objects are attached to
// the TestChain to enable cheat code functionality.
// Returns the tracer and associated pre-compile contracts, or an error, if one occurred.
func getCheatCodeProviders() (*cheatCodeTracer, []*cheatCodeContract, error) {
	// Create a cheat code tracer and attach it to the chain.
	tracer := &cheatCodeTracer{}

	// Obtain our cheat code pre-compiles
	hevmContract, err := getStandardCheatCodeContract(tracer)
	if err != nil {
		return nil, nil, err
	}

	// Return the tracer and precompiles
	return tracer, []*cheatCodeContract{hevmContract}, nil
}

// getStandardCheatCodeContract obtains a cheatCodeContract which implements common cheat codes.
// Returns the precompiled contract, or an error if one occurs.
func getStandardCheatCodeContract(tracer *cheatCodeTracer) (*cheatCodeContract, error) {
	// Define our address for this precompile contract, then create a new precompile to add methods to.
	contractAddress := common.HexToAddress("0x7109709ECfa91a80626fF3989D68f67F5b1DD12D")
	contract := newCheatCodeContract(tracer, contractAddress)

	// Define some basic ABI argument types
	uintType, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}

	// Add our contract methods
	err = contract.addMethod(
		"warp", abi.Arguments{{Type: uintType}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			// Set the vm context's time from the first input argument.
			tracer.evm.Context.Time.Set(inputs[0].(*big.Int))
			return nil, nil
		},
	)
	if err != nil {
		return nil, err
	}

	// Return our precompile contract information.
	return contract, nil
}
