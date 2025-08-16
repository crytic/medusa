package chain

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/crytic/medusa/chain/types"

	"github.com/crytic/medusa-geth/accounts/abi"
	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core/tracing"
	"github.com/crytic/medusa-geth/core/vm"
	"github.com/crytic/medusa-geth/crypto"
	"github.com/crytic/medusa/utils"
	"github.com/holiman/uint256"
)

// StandardCheatcodeContractAddress is the address for the standard cheatcode contract
var StandardCheatcodeContractAddress = common.HexToAddress("0x7109709ECfa91a80626fF3989D68f67F5b1DD12D")

// MaxUint64 holds the max value an uint64 can take
var _, MaxUint64 = utils.GetIntegerConstraints(false, 64)

// labelsKey describes the key to use when attempting to store and retrieve the chain's labels
const labelsKey = "Labels"

// GetLabels will return the labels attached to the transaction's messages results. Thus, every call sequence
// element will have access to all the labels that have been created until that point in time.
func GetLabels(messageResults *types.MessageResults) map[common.Address]string {
	// Try to obtain the results the tracer should've stored.
	if genericResult, ok := messageResults.AdditionalResults[labelsKey]; ok {
		if castedResult, ok := genericResult.(map[common.Address]string); ok {
			return castedResult
		}
	}

	// If we could not obtain them, return nil.
	return nil
}

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

	// Warp: Sets VM timestamp. Note that this _permanently_ updates the block timestamp for the remainder of the
	// chain's lifecycle.
	contract.addMethod(
		"warp", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Capture the original time
			originalTime := tracer.chain.pendingBlockContext.Time

			// Retrieve the new timestamp and make sure it is LEQ max value of an uint64
			newTime := inputs[0].(*big.Int)
			if newTime.Cmp(MaxUint64) > 0 {
				return nil, cheatCodeRevertData([]byte("warp: timestamp exceeds max value of type(uint64).max"))
			}

			// Set the time for the pending block context and the pending block
			// The block context will reflect the time change in the current EVM context
			// And the pending block time will allow for the new time to reflect
			// permanently for the remainder of the chain's existence.
			tracer.chain.pendingBlockContext.Time = newTime.Uint64()
			tracer.chain.pendingBlock.Header.Time = newTime.Uint64()

			// If the transaction reverts, we will restore the original time
			tracer.CurrentCallFrame().onChainRevertRestoreHooks.Push(func() {
				// The warp's effect will naturally revert if the chain reverts. Thus, we only want to handle the
				// case if the transaction that called warp reverts (which is why we have the nil checks).
				if tracer.chain.pendingBlockContext != nil {
					tracer.chain.pendingBlockContext.Time = originalTime
				}
				if tracer.chain.pendingBlock != nil {
					tracer.chain.pendingBlock.Header.Time = originalTime
				}
			})
			return nil, nil
		},
	)

	// Roll: Sets VM block number. Note that this _permanently_ updates the block number for the remainder of the
	// chain's lifecycle
	contract.addMethod(
		"roll", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Capture the original block number
			originalBlockNumber := tracer.chain.pendingBlockContext.BlockNumber

			// Retrieve the new block number
			newBlockNumber := inputs[0].(*big.Int)

			// Set the block number for the pending block context and the pending block
			// The block context will reflect the block number change in the current EVM context
			// And the pending block number will allow for the number to reflect
			// permanently for the remainder of the chain.
			tracer.chain.pendingBlockContext.BlockNumber.Set(newBlockNumber)
			tracer.chain.pendingBlock.Header.Number.Set(newBlockNumber)

			// If the transaction reverts, we will restore the original block number
			tracer.CurrentCallFrame().onChainRevertRestoreHooks.Push(func() {
				// The roll's effect will naturally revert if the chain reverts. Thus, we only want to handle the
				// case if the transaction that called roll reverts (which is why we have the nil checks).
				if tracer.chain.pendingBlockContext != nil {
					tracer.chain.pendingBlockContext.BlockNumber.Set(originalBlockNumber)
				}
				if tracer.chain.pendingBlock != nil {
					tracer.chain.pendingBlock.Header.Number.Set(originalBlockNumber)
				}
			})

			return nil, nil
		},
	)

	// Fee: Update the base fee. Note that this _permanently_ updates the base fee for the remainder of the
	// chain's lifecycle
	contract.addMethod(
		"fee", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Capture the original value
			original := new(big.Int).Set(tracer.chain.pendingBlockContext.BaseFee)

			// Update the pending block context and pending block's base fee
			// The block context will reflect the base fee change in the current EVM context
			// And the pending block will allow for the base fee to reflect
			// permanently for the remainder of the chain.
			tracer.chain.pendingBlockContext.BaseFee.Set(inputs[0].(*big.Int))
			tracer.chain.pendingBlock.Header.BaseFee.Set(inputs[0].(*big.Int))

			// If the transaction reverts, we will restore the original base fee
			tracer.CurrentCallFrame().onChainRevertRestoreHooks.Push(func() {
				// The fee's effect will naturally revert if the chain reverts. Thus, we only want to handle the
				// case if the transaction that called fee reverts (which is why we have the nil checks).
				if tracer.chain.pendingBlockContext != nil {
					tracer.chain.pendingBlockContext.BaseFee.Set(original)
				}
				if tracer.chain.pendingBlock != nil {
					tracer.chain.pendingBlock.Header.BaseFee.Set(original)
				}
			})
			return nil, nil
		},
	)

	// Difficulty: Updates difficulty. Since we do not allow users to choose the fork that
	// they are using (for now), and we are using a post-Paris fork, the difficulty cheatcode is a no-op.
	contract.addMethod(
		"difficulty", abi.Arguments{{Type: typeUint256}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			return nil, nil
		},
	)

	// Prevrandao: Updates random.
	contract.addMethod(
		"prevrandao", abi.Arguments{{Type: typeBytes32}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Store our original random
			originalRandom := tracer.chain.pendingBlockContext.Random

			// Update the pending block context random
			newRandom := inputs[0].([32]byte)
			newRandomHash := common.BytesToHash(newRandom[:])
			tracer.chain.pendingBlockContext.Random = &newRandomHash

			// Restore the original random when top frame exits
			tracer.CurrentCallFrame().onTopFrameExitRestoreHooks.Push(func() {
				tracer.chain.pendingBlockContext.Random = originalRandom
			})
			return nil, nil
		},
	)

	// Coinbase: Updates the block coinbase. Note that this _permanently_ updates the coinbase for the remainder of the
	// chain's lifecycle
	contract.addMethod(
		"coinbase", abi.Arguments{{Type: typeAddress}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Capture the original coinbase
			original := tracer.chain.pendingBlockContext.Coinbase

			// Update the pending block context and the pending block's coinbase
			// The block context will reflect the coinbase change in the current EVM context
			// And the pending block will allow for the coinbase change to reflect
			// permanently for the remainder of the chain.
			tracer.chain.pendingBlockContext.Coinbase = inputs[0].(common.Address)
			tracer.chain.pendingBlock.Header.Coinbase = inputs[0].(common.Address)

			// If the transaction reverts, we will restore the original base fee
			tracer.CurrentCallFrame().onChainRevertRestoreHooks.Push(func() {
				// The coinbase's effect will naturally revert if the chain reverts. Thus, we only want to handle the
				// case if the transaction that called coinbase reverts (which is why we have the nil checks).
				if tracer.chain.pendingBlockContext != nil {
					tracer.chain.pendingBlockContext.Coinbase = original
				}
				if tracer.chain.pendingBlock != nil {
					tracer.chain.pendingBlock.Header.Coinbase = original
				}
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

	// Label: Sets a label for an address.
	contract.addMethod(
		"label", abi.Arguments{{Type: typeAddress}, {Type: typeString}}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			addr := inputs[0].(common.Address)
			label := inputs[1].(string)
			tracer.chain.Labels[addr] = label
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
			tracer.chain.State().SetNonce(account, nonce, tracing.NonceChangeUnspecified)
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
				scopeContext.Contract.SetCaller(inputs[0].(common.Address))
				prankCallFrame.onFrameExitRestoreHooks.Push(func() {
					scopeContext.Contract.SetCaller(original)
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
			scopeContext.Contract.SetCaller(inputs[0].(common.Address))
			cheatCodeCallerFrame.onFrameExitRestoreHooks.Push(func() {
				scopeContext.Contract.SetCaller(original)
			})
			return nil, nil
		},
	)

	// Define our prank context and key mapping key to store it with.
	const extraDataKeyStartPrank = "cheatCodeStartPrankData"
	// StartPrankData holds the data for a startPrank operation.
	type StartPrankData struct {
		// enabled describes if this prank is currently active.
		enabled bool
		// msgSender describes the msg.sender to spoof for this prank.
		msgSender common.Address
		// txOrigin describes the tx.origin to spoof for this prank.
		// TODO: This is currently not used since we don't support spoofing of tx.origin.
		txOrigin *common.Address
		// delegateCall is an optional flag to handle delegatecalls.
		// TODO: This is currently not used since we don't support delegatecalls yet.
		delegateCall bool
		// setAtCallDepth describes the call depth at which this prank was set.
		setAtCallDepth uint64
	}

	// stopPrankFn holds the core functionality to stop an ongoing startPrank operation.
	stopPrankFn := func() {
		// Mark any `startPrank` data as disabled and delete it
		topLevelCallFrame := tracer.TopLevelCallFrame()
		phd := utils.MapFetchCasted[string, *StartPrankData](topLevelCallFrame.extraData, extraDataKeyStartPrank)
		if phd != nil {
			(*phd).enabled = false
			delete(topLevelCallFrame.extraData, extraDataKeyStartPrank)
		}
	}

	// startPrankFn holds the core functionality to prank the msg.sender for subsequent calls until stopPrank is called.
	startPrankFn := func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
		// Support all function prototypes by checking arg types and fetching appropriately.
		prankData := StartPrankData{
			enabled:        true,
			setAtCallDepth: tracer.callDepth - 1,
		}

		// Since the startPrank function has multiple overloads, we need to check the argument count and fetch appropriately.
		if len(inputs) == 1 {
			// 1 arg: argument is always the msg.sender prank address
			prankData.msgSender = inputs[0].(common.Address)
		} else if len(inputs) == 2 {
			// 2 args: the second is delegate call flag or tx.origin address.
			prankData.msgSender = inputs[0].(common.Address)
			if prankTxOrigin, ok := inputs[1].(common.Address); ok {
				prankData.txOrigin = &prankTxOrigin
			} else {
				prankData.delegateCall = inputs[1].(bool)
			}
		} else if len(inputs) == 3 {
			// 3 args: all of the above
			prankData.msgSender = inputs[0].(common.Address)
			prankTxOrigin := inputs[1].(common.Address)
			prankData.txOrigin = &prankTxOrigin
			prankData.delegateCall = inputs[2].(bool)
		} else {
			return nil, cheatCodeRevertData([]byte("startPrank: no overload supports this argument count"))
		}

		// Stop any existing prank as we'll be overwriting it.
		stopPrankFn()

		// Since startPrank/stopPrank don't act as a stack, we will simply set the prank data
		// across the whole transaction. So we'll store it within the top level call frame for easy access.
		topLevelCallFrame := tracer.TopLevelCallFrame()
		topLevelCallFrame.extraData[extraDataKeyStartPrank] = &prankData

		// Obtain the caller frame. This is a pre-compile, so we want to add an event to the frame which called us,
		// so when it enters the next frame in its scope, we trigger the prank.
		var hookFn func() = nil
		hookFn = func() {
			// If this prank was disabled but the hooks remain, do nothing until they clear out.
			if !prankData.enabled {
				return
			}

			// Obtain the call frame to prank
			prankCallFrame := tracer.CurrentCallFrame()

			// We can cast OpContext to ScopeContext because that is the type passed to OnOpcode.
			scopeContext := prankCallFrame.vmScope.(*vm.ScopeContext)

			// Determine if this is a delegatecall.
			isPrankedDelegateCall := false
			if prankData.delegateCall {
				if scopeContext.Contract.Address() != prankCallFrame.vmScope.Address() {
					isPrankedDelegateCall = true
				}
			}

			// Store the original value and spoof our prank address.
			originalMsgSender := scopeContext.Caller()
			scopeContext.Contract.SetCaller(prankData.msgSender)

			// Spoof the CodeAddress if we're pranking a delegatecall now.
			originalCodeAddress := scopeContext.Contract.Address()
			if isPrankedDelegateCall {
				// TODO: Prank CodeAddress.
				_ = originalCodeAddress // remove me (temp to avoid linter warnings)
				//  This is an addr field in `scopeContext.Contract.self`, which is unexposed in medusa-geth for now.
			}
			// TODO: Spoof tx.origin

			// Propagate the prank into next call in the same scope.
			if tracer.callDepth > prankData.setAtCallDepth {
				tracer.PreviousCallFrame().onNextFrameEnterHooks.Push(hookFn)
			}

			// Restore on exit (though pranking will continue for other frames)
			prankCallFrame.onFrameExitRestoreHooks.Push(func() {
				// If this prank was disabled but the hooks remain, do nothing until they clear out.
				if !prankData.enabled {
					return
				}

				// Restore msg.sender and, if delegatecall flag enabled, code address
				scopeContext.Contract.SetCaller(originalMsgSender)
				if isPrankedDelegateCall {
					// TODO: Restore CodeAddress
					_ = originalCodeAddress
				}

				// TODO: Restore tx.origin

				// If we exit the depth we started pranking at, remove the prank.
				if tracer.callDepth == prankData.setAtCallDepth {
					stopPrankFn()
				}
			})
		}
		tracer.PreviousCallFrame().onNextFrameEnterHooks.Push(hookFn)
		return nil, nil
	}

	// startPrank: Sets the msg.sender within all subsequent calls until stopPrank is called.
	contract.addMethod("startPrank", abi.Arguments{{Type: typeAddress}}, abi.Arguments{}, startPrankFn)
	// TODO: Implement `delegateCall` for startPrank
	//contract.addMethod("startPrank", abi.Arguments{{Type: typeAddress}, {Type: typeBool}}, abi.Arguments{}, startPrankFn)
	// TODO: Implement tx.origin for startPrank
	//contract.addMethod("startPrank", abi.Arguments{{Type: typeAddress}, {Type: typeAddress}}, abi.Arguments{}, startPrankFn)
	//contract.addMethod("startPrank", abi.Arguments{{Type: typeAddress}, {Type: typeAddress}, {Type: typeBool}}, abi.Arguments{}, startPrankFn)

	// stopPrank: Stops a startPrank operation.
	contract.addMethod(
		"stopPrank", abi.Arguments{}, abi.Arguments{},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			// Call the core stopPrankFn to disable any active prank.
			stopPrankFn()
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
			// Trim newlines, spaces, and the "0x" prefix
			trimmed := strings.TrimSpace(string(stdout))
			hexOut, err := hex.DecodeString(strings.TrimPrefix(trimmed, "0x"))
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
	// getCode: Retrieves the creation bytecode for a contract
	contract.addMethod("getCode", abi.Arguments{{Type: typeString}}, abi.Arguments{{Type: typeBytes}},
		func(tracer *cheatCodeTracer, inputs []any) ([]any, *cheatCodeRawReturnData) {
			contractPath := inputs[0].(string)

			_, contractName, err := parseContractPath(contractPath)
			if err != nil {
				return nil, cheatCodeRevertData(fmt.Appendf(nil, "getCode error: invalid path format: %v", err))
			}

			compiledContract, exists := tracer.chain.CompiledContracts[contractName]
			if !exists {
				return nil, cheatCodeRevertData(fmt.Appendf(nil, "getCode error: contract not found: %s", contractName))
			}

			bytecode := compiledContract.InitBytecode
			if len(bytecode) == 0 {
				return nil, cheatCodeRevertData(fmt.Appendf(nil, "getCode error: contract bytecode is empty: %s", contractName))
			}

			// Return the bytecode
			return []any{bytecode}, nil
		},
	)

	// Return our precompile contract information.
	return contract, nil
}

// parseContractPath parses a contract path in the following formats:
// - "MyContract.sol:MyContract"
// - "MyContract"
// Returns file name and contract name

func parseContractPath(path string) (string, string, error) {
	// Handle empty path
	if path == "" {
		return "", "", fmt.Errorf("empty path provided")
	}

	// Split by colon separator
	parts := strings.Split(path, ":")

	if len(parts) > 2 {
		return "", "", fmt.Errorf("too many path segments")
	}

	// When given "MyContract"
	if len(parts) == 1 && !strings.HasSuffix(parts[0], ".sol") {
		contractName := parts[0]
		fileName := contractName + ".sol"
		return fileName, contractName, nil
	}

	// When file specified: "MyContract.sol:MyContract" or "MyContract.sol"
	file := parts[0]
	if !strings.HasSuffix(file, ".sol") {
		file = file + ".sol"
	}

	// If contract name is explicitly provided after colon
	if len(parts) == 2 {
		return file, parts[1], nil
	}

	// Otherwise derive contract name from file name
	contractName := strings.TrimSuffix(filepath.Base(file), ".sol")
	return file, contractName, nil
}
