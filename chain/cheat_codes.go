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
	tracer := newCheatCodeTracer()

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
	typeAddress, err := abi.NewType("address", "", nil)
	if err != nil {
		return nil, err
	}
	typeBytes, err := abi.NewType("bytes", "", nil)
	if err != nil {
		return nil, err
	}
	typeBytes32, err := abi.NewType("bytes32", "", nil)
	if err != nil {
		return nil, err
	}
	typeUint64, err := abi.NewType("uint64", "", nil)
	if err != nil {
		return nil, err
	}
	typeUint256, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}

	// Warp: Sets VM timestamp
	contract.addMethod(
		"warp", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			// Maintain our changes until the transaction exits.
			original := new(big.Int).Set(tracer.evm.Context.Time)
			tracer.evm.Context.Time.Set(inputs[0].(*big.Int))
			tracer.TopCallFrame().onFrameExitHooks.Push(func() {
				tracer.evm.Context.Time.Set(original)
			})
			return nil, nil
		},
	)

	// Roll: Sets VM block number
	contract.addMethod(
		"roll", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			// Maintain our changes until the transaction exits.
			original := new(big.Int).Set(tracer.evm.Context.BlockNumber)
			tracer.evm.Context.BlockNumber.Set(inputs[0].(*big.Int))
			tracer.TopCallFrame().onFrameExitHooks.Push(func() {
				tracer.evm.Context.BlockNumber.Set(original)
			})
			return nil, nil
		},
	)

	// Roll: Sets VM block number
	contract.addMethod(
		"fee", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			// Maintain our changes until the transaction exits.
			original := new(big.Int).Set(tracer.evm.Context.BaseFee)
			tracer.evm.Context.BaseFee.Set(inputs[0].(*big.Int))
			tracer.TopCallFrame().onFrameExitHooks.Push(func() {
				tracer.evm.Context.BaseFee.Set(original)
			})
			return nil, nil
		},
	)

	// Difficulty: Sets VM block number
	contract.addMethod(
		"difficulty", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			// Maintain our changes until the transaction exits.

			// Obtain our spoofed difficulty
			spoofedDifficulty := inputs[0].(*big.Int)
			spoofedDifficultyHash := common.BigToHash(spoofedDifficulty)

			// Change difficulty in block context.
			originalDifficulty := new(big.Int).Set(tracer.evm.Context.Difficulty)
			tracer.evm.Context.Difficulty.Set(spoofedDifficulty)
			tracer.TopCallFrame().onFrameExitHooks.Push(func() {
				tracer.evm.Context.Difficulty.Set(originalDifficulty)
			})

			// In newer evm versions, block.difficulty uses opRandom instead of opDifficulty.
			// TODO: Check chain config here to see if the EVM version is 'Paris' or the consensus upgrade occurred.
			originalRandom := tracer.evm.Context.Random
			tracer.evm.Context.Random = &spoofedDifficultyHash
			tracer.TopCallFrame().onFrameExitHooks.Push(func() {
				tracer.evm.Context.Random = originalRandom
			})
			return nil, nil
		},
	)

	// ChainId: Sets VM chain ID
	contract.addMethod(
		"chainId", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			// Maintain our changes until the transaction exits.
			chainConfig := tracer.evm.ChainConfig()
			original := chainConfig.ChainID
			chainConfig.ChainID = inputs[0].(*big.Int)
			tracer.TopCallFrame().onFrameExitHooks.Push(func() {
				chainConfig.ChainID = original
			})
			return nil, nil
		},
	)

	// Store: Sets a storage slot value in a given account.
	contract.addMethod(
		"store", abi.Arguments{{Type: typeAddress}, {Type: typeBytes32}, {Type: typeBytes32}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			account := inputs[0].(common.Address)
			slot := inputs[1].([32]byte)
			value := inputs[2].([32]byte)
			tracer.evm.StateDB.SetState(account, slot, value)
			return nil, nil
		},
	)

	// Load: Loads a storage slot value from a given account.
	contract.addMethod(
		"load", abi.Arguments{{Type: typeAddress}, {Type: typeBytes32}}, abi.Arguments{{Type: typeBytes32}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			account := inputs[0].(common.Address)
			slot := inputs[1].([32]byte)
			value := tracer.evm.StateDB.GetState(account, slot)
			return []any{value}, nil
		},
	)

	// Etch: Sets the code for a given account.
	contract.addMethod(
		"etch", abi.Arguments{{Type: typeAddress}, {Type: typeBytes}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			account := inputs[0].(common.Address)
			code := inputs[1].([]byte)
			tracer.evm.StateDB.SetCode(account, code)
			return nil, nil
		},
	)

	// Deal: Sets the balance for a given account.
	contract.addMethod(
		"deal", abi.Arguments{{Type: typeAddress}, {Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			account := inputs[0].(common.Address)
			newBalance := inputs[1].(*big.Int)
			originalBalance := tracer.evm.StateDB.GetBalance(account)
			diff := new(big.Int).Sub(newBalance, originalBalance)
			tracer.evm.StateDB.AddBalance(account, diff)
			return nil, nil
		},
	)

	// GetNonce: Gets the nonce for a given account.
	contract.addMethod(
		"getNonce", abi.Arguments{{Type: typeAddress}}, abi.Arguments{{Type: typeUint64}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			account := inputs[0].(common.Address)
			nonce := tracer.evm.StateDB.GetNonce(account)
			return []any{nonce}, nil
		},
	)

	// SetNonce: Sets the nonce for a given account.
	contract.addMethod(
		"setNonce", abi.Arguments{{Type: typeAddress}, {Type: typeUint64}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			account := inputs[0].(common.Address)
			nonce := inputs[1].(uint64)
			tracer.evm.StateDB.SetNonce(account, nonce)
			return nil, nil
		},
	)

	// Coinbase: Sets the block coinbase.
	contract.addMethod(
		"coinbase", abi.Arguments{{Type: typeAddress}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			// Maintain our changes until the transaction exits.
			original := tracer.evm.Context.Coinbase
			tracer.evm.Context.Coinbase = inputs[0].(common.Address)
			tracer.TopCallFrame().onFrameExitHooks.Push(func() {
				tracer.evm.Context.Coinbase = original
			})
			return nil, nil
		},
	)

	// Prank: Sets the msg.sender within the next EVM call scope created by the caller.
	contract.addMethod(
		"prank", abi.Arguments{{Type: typeAddress}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			// Obtain the caller frame. This is a pre-compile, so we want to add an event to the frame which called us,
			// so when it enters the next frame in its scope, we trigger the prank.
			prankCallerFrame := tracer.PreviousCallFrame()
			prankCallerFrame.onNextFrameEnterHooks.Push(func() {
				// We entered the scope we want to prank, store the original value, patch, and add a hook to restore it
				// when this frame is exited.
				prankCallFrame := tracer.CurrentCallFrame()
				original := prankCallFrame.vmScope.Contract.CallerAddress
				prankCallFrame.vmScope.Contract.CallerAddress = inputs[0].(common.Address)
				tracer.CurrentCallFrame().onFrameExitHooks.Push(func() {
					prankCallFrame.vmScope.Contract.CallerAddress = original
				})
			})
			return nil, nil
		},
	)

	// PrankHere: Sets the msg.sender within caller EVM scope until it is exited.
	contract.addMethod(
		"prankHere", abi.Arguments{{Type: typeAddress}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, error) {
			// Obtain the caller frame. This is a pre-compile, so we want to add an event to the frame which called us,
			// so when it enters the next frame in its scope, we trigger the prank.
			prankCallerFrame := tracer.PreviousCallFrame()

			// Store the original value, patch, and add a hook to restore it when this frame is exited.
			original := prankCallerFrame.vmScope.Contract.CallerAddress
			prankCallerFrame.vmScope.Contract.CallerAddress = inputs[0].(common.Address)
			prankCallerFrame.onFrameExitHooks.Push(func() {
				prankCallerFrame.vmScope.Contract.CallerAddress = original
			})
			return nil, nil
		},
	)

	// Return our precompile contract information.
	return contract, nil
}
