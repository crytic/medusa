package chain

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/trailofbits/medusa/utils"
	"math/big"
	"os/exec"
	"strconv"
	"strings"
)

// getCheatCodeProviders obtains a cheatCodeTracer (used to power cheat code analysis) and associated CheatCodeContract
// objects linked to the tracer (providing on-chain callable methods as an entry point). These objects are attached to
// the TestChain to enable cheat code functionality.
// Returns the tracer and associated pre-compile contracts, or an error, if one occurred.
func getCheatCodeProviders() (*cheatCodeTracer, []*CheatCodeContract, error) {
	// Create a cheat code tracer and attach it to the chain.
	tracer := newCheatCodeTracer()

	// Obtain our cheat code pre-compiles
	stdCheatCodeContract, err := getStandardCheatCodeContract(tracer)
	if err != nil {
		return nil, nil, err
	}

	// Return the tracer and precompiles
	return tracer, []*CheatCodeContract{stdCheatCodeContract}, nil
}

// getStandardCheatCodeContract obtains a CheatCodeContract which implements common cheat codes.
// Returns the precompiled contract, or an error if one occurs.
func getStandardCheatCodeContract(tracer *cheatCodeTracer) (*CheatCodeContract, error) {
	// Define our address for this precompile contract, then create a new precompile to add methods to.
	contractAddress := common.HexToAddress("0x7109709ECfa91a80626fF3989D68f67F5b1DD12D")
	contract := newCheatCodeContract(tracer, contractAddress, "StdCheats")

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
	typeUint8, err := abi.NewType("uint8", "", nil)
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
	typeInt256, err := abi.NewType("int256", "", nil)
	if err != nil {
		return nil, err
	}
	typeStringSlice, err := abi.NewType("string[]", "", nil)
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

	// Warp: Sets VM timestamp
	contract.addMethod(
		"warp", abi.Arguments{{Type: typeUint64}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Maintain our changes until the transaction exits.
			original := tracer.evm.Context.Time
			tracer.evm.Context.Time = inputs[0].(uint64)
			tracer.CurrentCallFrame().onTopFrameExitRestoreHooks.Push(func() {
				tracer.evm.Context.Time = original
			})
			return nil, nil
		},
	)

	// Roll: Sets VM block number
	contract.addMethod(
		"roll", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Maintain our changes until the transaction exits.
			original := new(big.Int).Set(tracer.evm.Context.BlockNumber)
			tracer.evm.Context.BlockNumber.Set(inputs[0].(*big.Int))
			tracer.CurrentCallFrame().onTopFrameExitRestoreHooks.Push(func() {
				tracer.evm.Context.BlockNumber.Set(original)
			})
			return nil, nil
		},
	)

	// Fee: Update base fee
	contract.addMethod(
		"fee", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Maintain our changes until the transaction exits.
			original := new(big.Int).Set(tracer.evm.Context.BaseFee)
			tracer.evm.Context.BaseFee.Set(inputs[0].(*big.Int))
			tracer.CurrentCallFrame().onTopFrameExitRestoreHooks.Push(func() {
				tracer.evm.Context.BaseFee.Set(original)
			})
			return nil, nil
		},
	)

	// Difficulty: Sets VM block number
	contract.addMethod(
		"difficulty", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Maintain our changes until the transaction exits.

			// Obtain our spoofed difficulty
			spoofedDifficulty := inputs[0].(*big.Int)
			spoofedDifficultyHash := common.BigToHash(spoofedDifficulty)

			// Change difficulty in block context.
			originalDifficulty := new(big.Int).Set(tracer.evm.Context.Difficulty)
			tracer.evm.Context.Difficulty.Set(spoofedDifficulty)
			tracer.CurrentCallFrame().onTopFrameExitRestoreHooks.Push(func() {
				tracer.evm.Context.Difficulty.Set(originalDifficulty)
			})

			// In newer evm versions, block.difficulty uses opRandom instead of opDifficulty.
			// TODO: Check chain config here to see if the EVM version is 'Paris' or the consensus upgrade occurred.
			originalRandom := tracer.evm.Context.Random
			tracer.evm.Context.Random = &spoofedDifficultyHash
			tracer.CurrentCallFrame().onTopFrameExitRestoreHooks.Push(func() {
				tracer.evm.Context.Random = originalRandom
			})
			return nil, nil
		},
	)

	// ChainId: Sets VM chain ID
	contract.addMethod(
		"chainId", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Maintain our changes unless this code path reverts or the whole transaction is reverted in the chain.
			chainConfig := tracer.evm.ChainConfig()
			original := chainConfig.ChainID
			chainConfig.ChainID = inputs[0].(*big.Int)
			tracer.CurrentCallFrame().onChainRevertRestoreHooks.Push(func() {
				chainConfig.ChainID = original
			})
			return nil, nil
		},
	)

	// Store: Sets a storage slot value in a given account.
	contract.addMethod(
		"store", abi.Arguments{{Type: typeAddress}, {Type: typeBytes32}, {Type: typeBytes32}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
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
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			account := inputs[0].(common.Address)
			slot := inputs[1].([32]byte)
			value := tracer.evm.StateDB.GetState(account, slot)
			return []any{value}, nil
		},
	)

	// Etch: Sets the code for a given account.
	contract.addMethod(
		"etch", abi.Arguments{{Type: typeAddress}, {Type: typeBytes}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			account := inputs[0].(common.Address)
			code := inputs[1].([]byte)
			tracer.evm.StateDB.SetCode(account, code)
			return nil, nil
		},
	)

	// Deal: Sets the balance for a given account.
	contract.addMethod(
		"deal", abi.Arguments{{Type: typeAddress}, {Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
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
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			account := inputs[0].(common.Address)
			nonce := tracer.evm.StateDB.GetNonce(account)
			return []any{nonce}, nil
		},
	)

	// SetNonce: Sets the nonce for a given account.
	contract.addMethod(
		"setNonce", abi.Arguments{{Type: typeAddress}, {Type: typeUint64}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			account := inputs[0].(common.Address)
			nonce := inputs[1].(uint64)
			tracer.evm.StateDB.SetNonce(account, nonce)
			return nil, nil
		},
	)

	// Coinbase: Sets the block coinbase.
	contract.addMethod(
		"coinbase", abi.Arguments{{Type: typeAddress}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Maintain our changes until the transaction exits.
			original := tracer.evm.Context.Coinbase
			tracer.evm.Context.Coinbase = inputs[0].(common.Address)
			tracer.CurrentCallFrame().onTopFrameExitRestoreHooks.Push(func() {
				tracer.evm.Context.Coinbase = original
			})
			return nil, nil
		},
	)

	// Prank: Sets the msg.sender within the next EVM call scope created by the caller.
	contract.addMethod(
		"prank", abi.Arguments{{Type: typeAddress}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Obtain the caller frame. This is a pre-compile, so we want to add an event to the frame which called us,
			// so when it enters the next frame in its scope, we trigger the prank.
			cheatCodeCallerFrame := tracer.PreviousCallFrame()
			cheatCodeCallerFrame.onNextFrameEnterHooks.Push(func() {
				// We entered the scope we want to prank, store the original value, patch, and add a hook to restore it
				// when this frame is exited.
				prankCallFrame := tracer.CurrentCallFrame()
				original := prankCallFrame.vmScope.Contract.CallerAddress
				prankCallFrame.vmScope.Contract.CallerAddress = inputs[0].(common.Address)
				prankCallFrame.onFrameExitRestoreHooks.Push(func() {
					prankCallFrame.vmScope.Contract.CallerAddress = original
				})
			})
			return nil, nil
		},
	)

	// PrankHere: Sets the msg.sender within caller EVM scope until it is exited.
	contract.addMethod(
		"prankHere", abi.Arguments{{Type: typeAddress}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Obtain the caller frame. This is a pre-compile, so we want to add an event to the frame which called us,
			// to disable the cheat code on exit
			cheatCodeCallerFrame := tracer.PreviousCallFrame()

			// Store the original value, patch, and add a hook to restore it when this frame is exited.
			original := cheatCodeCallerFrame.vmScope.Contract.CallerAddress
			cheatCodeCallerFrame.vmScope.Contract.CallerAddress = inputs[0].(common.Address)
			cheatCodeCallerFrame.onFrameExitRestoreHooks.Push(func() {
				cheatCodeCallerFrame.vmScope.Contract.CallerAddress = original
			})
			return nil, nil
		},
	)

	// FFI: Run arbitrary command on base OS
	contract.addMethod(
		"ffi", abi.Arguments{{Type: typeStringSlice}}, abi.Arguments{{Type: typeBytes}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Ensure FFI is enabled (this allows arbitrary code execution, so we expect it to be explicitly enabled).
			if !tracer.chain.testChainConfig.CheatCodeConfig.EnableFFI {
				// Make sure there is at least a command to run
				return nil, cheatCodeRevertData([]byte("ffi is not enabled in the chain configuration"))
			}

			// command is cmdAndInputs[0] and args are cmdAndInputs[1:]
			cmdAndInputs := inputs[0].([]string)

			var command string
			var args []string

			if len(cmdAndInputs) < 1 {
				// Make sure there is at least a command to run
				return nil, cheatCodeRevertData([]byte("ffi: no command was provided"))
			} else if len(cmdAndInputs) == 1 {
				// It is possible there are no arguments provided
				command = cmdAndInputs[0]
			} else {
				// Both a command and arguments have been provided
				command = cmdAndInputs[0]
				args = cmdAndInputs[1:]
			}

			// Create our command
			cmd := exec.Command(command, args...)

			// Execute it and grab the output
			stdout, _, combined, err := utils.RunCommandWithOutputAndError(cmd)
			if err != nil {
				errorMsg := fmt.Sprintf("ffi: cmd failed with the following error: %v\nOutput: %v", err, string(combined))
				return nil, cheatCodeRevertData([]byte(errorMsg))
			}

			// Attempt to hex decode the output
			hexOut, err := hex.DecodeString(strings.TrimPrefix(string(stdout), "0x"))
			if err != nil {
				// Return the byte array as itself if hex decoding does not work
				return []any{stdout}, nil
			}

			// Hex decoding worked, so return that
			return []any{hexOut}, nil
		},
	)

	// addr: Compute the address for a given private key
	contract.addMethod("addr", abi.Arguments{{Type: typeUint256}}, abi.Arguments{{Type: typeAddress}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Using TOECDSAUnsafe b/c the private key is guaranteed to be of length 256 bits, at most
			privateKey := crypto.ToECDSAUnsafe(inputs[0].(*big.Int).Bytes())

			// Get ECDSA public key
			publicKey := privateKey.Public().(*ecdsa.PublicKey)

			// Get associated address
			addr := crypto.PubkeyToAddress(*publicKey)

			return []any{addr}, nil
		},
	)

	// sign: Sign a digest given some private key
	contract.addMethod("sign", abi.Arguments{{Type: typeUint256}, {Type: typeBytes32}},
		abi.Arguments{{Type: typeUint8}, {Type: typeBytes32}, {Type: typeBytes32}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Using TOECDSAUnsafe b/c the private key is guaranteed to be of length 256 bits, at most
			privateKey := crypto.ToECDSAUnsafe(inputs[0].(*big.Int).Bytes())
			digest := inputs[1].([32]byte)

			// Sign digest
			sig, err := crypto.Sign(digest[:], privateKey)
			if err != nil {
				return nil, cheatCodeRevertData([]byte("sign: malformed input to signature algorithm"))
			}

			// `r` and `s` have to be [32]byte arrays
			var r [32]byte
			var s [32]byte
			copy(r[:], sig[:32])
			copy(s[:], sig[32:64])

			// Need to add 27 to the `v` value for ecrecover to work
			v := sig[64] + 27

			return []any{v, r, s}, nil
		},
	)

	// toString(address): Convert address to string
	contract.addMethod("toString", abi.Arguments{{Type: typeAddress}}, abi.Arguments{{Type: typeString}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			addr := inputs[0].(common.Address)
			return []any{addr.String()}, nil
		},
	)

	// toString(bool): Convert bool to string
	contract.addMethod("toString", abi.Arguments{{Type: typeBool}}, abi.Arguments{{Type: typeString}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			b := inputs[0].(bool)
			return []any{strconv.FormatBool(b)}, nil
		},
	)

	// toString(uint256): Convert uint256 to string
	contract.addMethod("toString", abi.Arguments{{Type: typeUint256}}, abi.Arguments{{Type: typeString}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			n := inputs[0].(*big.Int)
			return []any{n.String()}, nil
		},
	)

	// toString(int256): Convert int256 to string
	contract.addMethod("toString", abi.Arguments{{Type: typeInt256}}, abi.Arguments{{Type: typeString}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			n := inputs[0].(*big.Int)
			return []any{n.String()}, nil
		},
	)

	// toString(bytes32): Convert bytes32 to string
	contract.addMethod("toString", abi.Arguments{{Type: typeBytes32}}, abi.Arguments{{Type: typeString}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			b := inputs[0].([32]byte)
			// Prefix "0x"
			hexString := "0x" + hex.EncodeToString(b[:])

			return []any{hexString}, nil
		},
	)

	// toString(bytes): Convert bytes to string
	contract.addMethod("toString", abi.Arguments{{Type: typeBytes}}, abi.Arguments{{Type: typeString}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			b := inputs[0].([]byte)
			// Prefix "0x"
			hexString := "0x" + hex.EncodeToString(b)

			return []any{hexString}, nil
		},
	)

	// parseBytes: Convert string to bytes
	contract.addMethod("parseBytes", abi.Arguments{{Type: typeString}}, abi.Arguments{{Type: typeBytes}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			return []any{[]byte(inputs[0].(string))}, nil
		},
	)

	// parseBytes32: Convert string to bytes32
	contract.addMethod("parseBytes32", abi.Arguments{{Type: typeString}}, abi.Arguments{{Type: typeBytes32}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			bSlice := []byte(inputs[0].(string))

			// Use a fixed array and copy the data over
			var bArray [32]byte
			copy(bArray[:], bSlice[:32])

			return []any{bArray}, nil
		},
	)

	// parseAddress: Convert string to address
	contract.addMethod("parseAddress", abi.Arguments{{Type: typeString}}, abi.Arguments{{Type: typeAddress}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			addr, err := utils.HexStringToAddress(inputs[0].(string))
			if err != nil {
				return nil, cheatCodeRevertData([]byte("parseAddress: malformed string"))
			}

			return []any{addr}, nil
		},
	)

	// parseUint: Convert string to uint256
	contract.addMethod("parseUint", abi.Arguments{{Type: typeString}}, abi.Arguments{{Type: typeUint256}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			n, ok := new(big.Int).SetString(inputs[0].(string), 10)
			if !ok {
				return nil, cheatCodeRevertData([]byte("parseUint: malformed string"))
			}

			return []any{n}, nil
		},
	)

	// parseInt: Convert string to int256
	contract.addMethod("parseInt", abi.Arguments{{Type: typeString}}, abi.Arguments{{Type: typeInt256}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			n, ok := new(big.Int).SetString(inputs[0].(string), 10)
			if !ok {
				return nil, cheatCodeRevertData([]byte("parseInt: malformed string"))
			}

			return []any{n}, nil
		},
	)

	// parseBool: Convert string to bool
	contract.addMethod("parseBool", abi.Arguments{{Type: typeString}}, abi.Arguments{{Type: typeBool}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			b, err := strconv.ParseBool(inputs[0].(string))
			if err != nil {
				return nil, cheatCodeRevertData([]byte("parseBool: malformed string"))
			}

			return []any{b}, nil
		},
	)

	// Return our precompile contract information.
	return contract, nil
}
