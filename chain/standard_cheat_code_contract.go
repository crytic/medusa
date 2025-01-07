package chain

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/crytic/medusa/compilation/abiutils"
	"github.com/crytic/medusa/logging"
	"math/big"
	"os/exec"
	"strconv"
	"strings"

	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// StandardCheatcodeContractAddress is the address for the standard cheatcode contract
var StandardCheatcodeContractAddress = common.HexToAddress("0x7109709ECfa91a80626fF3989D68f67F5b1DD12D")

// MaxUint64 holds the max value an uint64 can take
var _, MaxUint64 = utils.GetIntegerConstraints(false, 64)

// getStandardCheatCodeContract obtains a CheatCodeContract which implements common cheat codes.
// Returns the precompiled contract, or an error if one occurs.
func getStandardCheatCodeContract(tracer *cheatCodeTracer) (*CheatCodeContract, error) {
	// Create a new precompile to add methods to.
	contract := newCheatCodeContract(tracer, StandardCheatcodeContractAddress, "StdCheats")

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
		"warp", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Maintain our changes until the transaction exits.
			originalTime := tracer.chain.pendingBlockContext.Time

			// Retrieve new timestamp and make sure it is LEQ max value of an uint64
			newTime := inputs[0].(*big.Int)
			if newTime.Cmp(MaxUint64) > 0 {
				return nil, cheatCodeRevertData([]byte("warp: timestamp exceeds max value of type(uint64).max"))
			}

			// Set the time
			tracer.chain.pendingBlockContext.Time = newTime.Uint64()
			tracer.CurrentCallFrame().onTopFrameExitRestoreHooks.Push(func() {
				// Reset the time
				tracer.chain.pendingBlockContext.Time = originalTime
			})
			return nil, nil
		},
	)

	// Roll: Sets VM block number
	contract.addMethod(
		"roll", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Maintain our changes until the transaction exits.
			original := new(big.Int).Set(tracer.chain.pendingBlockContext.BlockNumber)
			tracer.chain.pendingBlockContext.BlockNumber.Set(inputs[0].(*big.Int))
			tracer.CurrentCallFrame().onTopFrameExitRestoreHooks.Push(func() {
				tracer.chain.pendingBlockContext.BlockNumber.Set(original)
			})
			return nil, nil
		},
	)

	// Fee: Update base fee
	contract.addMethod(
		"fee", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Maintain our changes until the transaction exits.
			original := new(big.Int).Set(tracer.chain.pendingBlockContext.BaseFee)
			tracer.chain.pendingBlockContext.BaseFee.Set(inputs[0].(*big.Int))
			tracer.CurrentCallFrame().onTopFrameExitRestoreHooks.Push(func() {
				tracer.chain.pendingBlockContext.BaseFee.Set(original)
			})
			return nil, nil
		},
	)

	// Difficulty: Updates difficulty
	contract.addMethod(
		"difficulty", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Maintain our changes until the transaction exits.
			spoofedDifficulty := inputs[0].(*big.Int)
			spoofedDifficultyHash := common.BigToHash(spoofedDifficulty)
			originalRandom := tracer.chain.pendingBlockContext.Random

			// In newer evm versions, block.difficulty uses opRandom instead of opDifficulty.
			tracer.chain.pendingBlockContext.Random = &spoofedDifficultyHash
			tracer.CurrentCallFrame().onTopFrameExitRestoreHooks.Push(func() {
				tracer.chain.pendingBlockContext.Random = originalRandom
			})
			return nil, nil
		},
	)

	// ChainId: Sets VM chain ID
	contract.addMethod(
		"chainId", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Maintain our changes unless this code path reverts or the whole transaction is reverted in the chain.
			chainConfig := tracer.chain.pendingBlockChainConfig
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
			tracer.chain.State().SetState(account, slot, value)
			return nil, nil
		},
	)

	// Load: Loads a storage slot value from a given account.
	contract.addMethod(
		"load", abi.Arguments{{Type: typeAddress}, {Type: typeBytes32}}, abi.Arguments{{Type: typeBytes32}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			account := inputs[0].(common.Address)
			slot := inputs[1].([32]byte)
			value := tracer.chain.State().GetState(account, slot)
			return []any{value}, nil
		},
	)

	// Etch: Sets the code for a given account.
	contract.addMethod(
		"etch", abi.Arguments{{Type: typeAddress}, {Type: typeBytes}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			account := inputs[0].(common.Address)
			code := inputs[1].([]byte)
			tracer.chain.State().SetCode(account, code)
			return nil, nil
		},
	)

	// Deal: Sets the balance for a given account.
	contract.addMethod(
		"deal", abi.Arguments{{Type: typeAddress}, {Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			account := inputs[0].(common.Address)
			newBalance := inputs[1].(*big.Int)
			newBalanceUint256 := new(uint256.Int)
			newBalanceUint256.SetFromBig(newBalance)
			tracer.chain.State().SetBalance(account, newBalanceUint256, tracing.BalanceChangeUnspecified)
			return nil, nil
		},
	)

	// GetNonce: Gets the nonce for a given account.
	contract.addMethod(
		"getNonce", abi.Arguments{{Type: typeAddress}}, abi.Arguments{{Type: typeUint64}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			account := inputs[0].(common.Address)
			nonce := tracer.chain.State().GetNonce(account)
			return []any{nonce}, nil
		},
	)

	// SetNonce: Sets the nonce for a given account.
	contract.addMethod(
		"setNonce", abi.Arguments{{Type: typeAddress}, {Type: typeUint64}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			account := inputs[0].(common.Address)
			nonce := inputs[1].(uint64)
			tracer.chain.State().SetNonce(account, nonce)
			return nil, nil
		},
	)

	// Coinbase: Sets the block coinbase.
	contract.addMethod(
		"coinbase", abi.Arguments{{Type: typeAddress}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Maintain our changes until the transaction exits.
			original := tracer.chain.pendingBlockContext.Coinbase
			tracer.chain.pendingBlockContext.Coinbase = inputs[0].(common.Address)
			tracer.CurrentCallFrame().onTopFrameExitRestoreHooks.Push(func() {
				tracer.chain.pendingBlockContext.Coinbase = original
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
				// We can cast OpContext to ScopeContext because that is the type passed to OnOpcode.
				scopeContext := prankCallFrame.vmScope.(*vm.ScopeContext)
				original := scopeContext.Caller()
				scopeContext.Contract.CallerAddress = inputs[0].(common.Address)
				prankCallFrame.onFrameExitRestoreHooks.Push(func() {
					scopeContext.Contract.CallerAddress = original
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
			// We can cast OpContext to ScopeContext because that is the type passed to OnOpcode.
			scopeContext := cheatCodeCallerFrame.vmScope.(*vm.ScopeContext)
			original := scopeContext.Caller()
			scopeContext.Contract.CallerAddress = inputs[0].(common.Address)
			cheatCodeCallerFrame.onFrameExitRestoreHooks.Push(func() {
				scopeContext.Contract.CallerAddress = original
			})
			return nil, nil
		},
	)

	// snapshot: Takes a snapshot of the current state of the evm and returns the id associated with the snapshot
	contract.addMethod(
		"snapshot", abi.Arguments{}, abi.Arguments{{Type: typeUint256}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			snapshotID := tracer.chain.State().Snapshot()

			return []any{snapshotID}, nil
		},
	)

	// revertTo(uint256): Revert the state of the evm to a previous snapshot. Takes the snapshot id to revert to.
	contract.addMethod(
		"revertTo", abi.Arguments{{Type: typeUint256}}, abi.Arguments{{Type: typeBool}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			snapshotID := inputs[0].(*big.Int)
			tracer.chain.State().RevertToSnapshot(int(snapshotID.Int64()))

			return []any{true}, nil
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
			// Get the private key object
			privateKey, err := utils.GetPrivateKey(inputs[0].(*big.Int).Bytes())
			if err != nil {
				errorMessage := "addr: " + err.Error()
				return nil, cheatCodeRevertData([]byte(errorMessage))
			}

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
			// Get the private key object
			privateKey, err := utils.GetPrivateKey(inputs[0].(*big.Int).Bytes())
			if err != nil {
				errorMessage := "sign: " + err.Error()
				return nil, cheatCodeRevertData([]byte(errorMessage))
			}

			// Sign digest
			digest := inputs[1].([32]byte)
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
			copy(bArray[:], bSlice)

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

	// expectEmit() with no arguments - check all topics and data
	contract.addMethod(
		"expectEmit", abi.Arguments{}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			return expectEmitHandler(tracer, true, true, true, true, nil)
		},
	)

	// expectEmit(bool,bool,bool,bool) - specify which elements to check
	contract.addMethod(
		"expectEmit", abi.Arguments{
			{Type: typeBool}, // checkTopic1
			{Type: typeBool}, // checkTopic2
			{Type: typeBool}, // checkTopic3  
			{Type: typeBool}, // checkData
		}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			return expectEmitHandler(tracer,
				inputs[0].(bool), // checkTopic1
				inputs[1].(bool), // checkTopic2 
				inputs[2].(bool), // checkTopic3
				inputs[3].(bool), // checkData
				nil,             // emitter
			)
		},
	)

	// expectEmit(address) - check all topics and data from specific emitter
	contract.addMethod(
		"expectEmit", abi.Arguments{{Type: typeAddress}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			return expectEmitHandler(tracer, true, true, true, true, inputs[0].(common.Address))
		},
	)

	// expectEmit(bool,bool,bool,bool,address) - specify elements to check and emitter
	contract.addMethod(
		"expectEmit", abi.Arguments{
			{Type: typeBool},    // checkTopic1
			{Type: typeBool},    // checkTopic2
			{Type: typeBool},    // checkTopic3
			{Type: typeBool},    // checkData
			{Type: typeAddress}, // emitter
		}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			return expectEmitHandler(tracer,
				inputs[0].(bool), // checkTopic1
				inputs[1].(bool), // checkTopic2
				inputs[2].(bool), // checkTopic3
				inputs[3].(bool), // checkData
				inputs[4].(common.Address), // emitter
			)
		},
	)

	// Return our precompile contract information.
	return contract, nil
}

// expectEmitHandler handles the implementation of the expectEmit cheatcode variants
func expectEmitHandler(tracer *cheatCodeTracer, checkTopic1, checkTopic2, checkTopic3, checkData bool, emitter common.Address) ([]any, *cheatCodeRawReturnData) {
	// Get the caller frame
	cheatCodeCallerFrame := tracer.PreviousCallFrame()
	
	// Initialize sub logger
	logger := logging.GlobalLogger.NewSubLogger("module", "cheatcodes")

	// Flag to track if we've seen the expected event
	enteredNewCallframe := false
	expectedEventSeen := false

	// We need to grab the expected log from the next event emitted after this cheat call
	var expectedLog *types.Log

	// Attach hook for the next log emission 
	var expectEmitHook func()
	expectEmitHook = func() {
		currentFrame := tracer.CurrentCallFrame()

		// Skip cheatcode calls
		if currentFrame.vmScope == nil {
			cheatCodeCallerFrame.onNextFrameEnterHooks.Push(expectEmitHook)
			return
		}

		// Update flag if not already updated to indicate that a new callframe was entered
		if !enteredNewCallframe {
			enteredNewCallframe = true
		}

		// Watch for LOG0-LOG4 opcodes
		if currentFrame.vmOp >= vm.LOG0 && currentFrame.vmOp <= vm.LOG4 {
			logs := tracer.chain.State().Logs()
			if len(logs) > 0 {
				lastLog := logs[len(logs)-1]

				// If this is our first log after setting up expectEmit, save it as our expected log
				if expectedLog == nil {
					expectedLog = lastLog
					return
				}

				// For subsequent logs, verify they match what we expect
				if emitter != common.Address{} && lastLog.Address != emitter {
					return // Skip logs from other emitters
				}

				// Always check topic[0] (event signature)
				if len(lastLog.Topics) > 0 && len(expectedLog.Topics) > 0 {
					if lastLog.Topics[0] == expectedLog.Topics[0] {
						matched := true

						// Check remaining topics based on flags
						if checkTopic1 && len(lastLog.Topics) > 1 && len(expectedLog.Topics) > 1 {
							matched = matched && (lastLog.Topics[1] == expectedLog.Topics[1])
						}
						if checkTopic2 && len(lastLog.Topics) > 2 && len(expectedLog.Topics) > 2 {
							matched = matched && (lastLog.Topics[2] == expectedLog.Topics[2]) 
						}
						if checkTopic3 && len(lastLog.Topics) > 3 && len(expectedLog.Topics) > 3 {
							matched = matched && (lastLog.Topics[3] == expectedLog.Topics[3])
						}

						// Check log data if requested
						if checkData {
							matched = matched && bytes.Equal(lastLog.Data, expectedLog.Data)
						}

						if matched {
							expectedEventSeen = true
						}
					}
				}
			}
		}
	}

	// Add hook for next frame
	cheatCodeCallerFrame.onNextFrameEnterHooks.Push(expectEmitHook)

	// Check if we found the expected event when exiting
	cheatCodeCallerFrame.onTopFrameExitRestoreHooks.Push(func() {
		if !enteredNewCallframe {
			logger.Error("expectEmit: No event emitted")
			tracer.ThrowAssertionError()
		}
		if !expectedEventSeen {
			logger.Error("expectEmit: Event did not match expected event")
			tracer.ThrowAssertionError()
		}
	})

	return nil, nil
}
