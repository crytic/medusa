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
	stdCheatCodeContract, err := getStandardCheatCodeContract(tracer)
	if err != nil {
		return nil, nil, err
	}

	// Return the tracer and precompiles
	return tracer, []*cheatCodeContract{stdCheatCodeContract}, nil
}

// getStandardCheatCodeContract obtains a cheatCodeContract which implements common cheat codes.
// Returns the precompiled contract, or an error if one occurs.
func getStandardCheatCodeContract(tracer *cheatCodeTracer) (*cheatCodeContract, error) {
	// Define our address for this precompile contract, then create a new precompile to add methods to.
	contractAddress := common.HexToAddress("0x7109709ECfa91a80626fF3989D68f67F5b1DD12D")
	contract := newCheatCodeContract(tracer, contractAddress)

	// Define some basic ABI argument types
	typeUint256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}

	// Warp: Sets VM timestamp
	contract.addMethod(
		"warp", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			// Set the vm context's time from the first input argument.
			tracer.evm.Context.Time.Set(inputs[0].(*big.Int))
			return nil, nil
		},
	)

	// Roll: Sets VM block number
	contract.addMethod(
		"roll", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			// Set the vm context's block number from the first input argument.
			tracer.evm.Context.BlockNumber.Set(inputs[0].(*big.Int))
			return nil, nil
		},
	)

	// Roll: Sets VM block number
	contract.addMethod(
		"fee", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			// Set the vm context's base fee from the first input argument.
			tracer.evm.Context.BaseFee.Set(inputs[0].(*big.Int))
			return nil, nil
		},
	)

	// Difficulty: Sets VM block number
	contract.addMethod(
		"difficulty", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			// Set the vm context's difficulty from the first input argument.
			tracer.evm.Context.Difficulty.Set(inputs[0].(*big.Int))

			// In newer evm versions, block.difficulty uses opRandom instead of opDifficulty.
			// TODO: Check chain config here to see if the EVM version is 'Paris' or the consensus upgrade occurred.
			hash := common.BigToHash(inputs[0].(*big.Int))
			tracer.evm.Context.Random = &hash
			return nil, nil
		},
	)

	// ChainId: Sets VM chain ID
	contract.addMethod(
		"chainId", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			// Set the vm chain config's chain id from the first input argument.
			tracer.evm.ChainConfig().ChainID.Set(inputs[0].(*big.Int))

			// TODO: Before we enable this, we need to verify this is not the same ChainConfig supplied from the
			//  TestChain, as this may stick across a chain revert (during the fuzzing loop).
			//  If so, we will want to store the original value and patch/restore it every tx start/end.
			panic("not fully implemented")
			return nil, nil
		},
	)

	// Return our precompile contract information.
	return contract, nil
}
