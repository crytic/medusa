// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vendored

import (
	"math/big"

	"github.com/crytic/medusa-geth/common"
	. "github.com/crytic/medusa-geth/core"
	gethtypes "github.com/crytic/medusa-geth/core/types"
	"github.com/crytic/medusa-geth/core/vm"
	"github.com/crytic/medusa-geth/crypto"
	"github.com/crytic/medusa-geth/params"
	"github.com/crytic/medusa/chain/config"
	"github.com/crytic/medusa/chain/types"
)

// EVMApplyTransaction is a vendored version of go-ethereum's unexported applyTransaction method (not to be confused
// with the exported method ApplyTransaction). This method was vendored to simply be exposed/exported, so it can be
// used by the test chain. This method offers greater control of parameters over the exposed ApplyTransaction
// method. Its purpose is to take a message (internal transaction) and apply state transition updates to our current
// state as if we had just previously received and validated a transaction which the message was derived from.
// This executes on an underlying EVM and returns a transaction receipt, or an error if one occurs.
// Additional changes:
// - Exposed core.ExecutionResult as a return value.
func EVMApplyTransaction(msg *Message, config *params.ChainConfig, testChainConfig *config.TestChainConfig, author *common.Address, gp *GasPool, statedb types.MedusaStateDB, blockNumber *big.Int, blockHash common.Hash, tx *gethtypes.Transaction, usedGas *uint64, evm *vm.EVM) (receipt *gethtypes.Receipt, result *ExecutionResult, err error) {
	// Apply the OnTxStart and OnTxEnd hooks
	if evm.Config.Tracer != nil && evm.Config.Tracer.OnTxStart != nil {
		evm.Config.Tracer.OnTxStart(evm.GetVMContext(), tx, msg.From)
		if evm.Config.Tracer.OnTxEnd != nil {
			defer func() {
				evm.Config.Tracer.OnTxEnd(receipt, err)
			}()
		}
	}

	// Apply the transaction to the current state (included in the env).
	result, err = ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, nil, err
	}

	// Update the state with pending changes.
	var root []byte
	if config.IsByzantium(blockNumber) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(blockNumber)).Bytes()
	}
	*usedGas += result.UsedGas

	// TODO: Explore using the MakeReceipt function in `core`. The core risk is an interface conversion which will have
	//  a potential perf hit
	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt = &gethtypes.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: *usedGas}
	if result.Failed() {
		receipt.Status = gethtypes.ReceiptStatusFailed
	} else {
		receipt.Status = gethtypes.ReceiptStatusSuccessful
	}
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas

	// If the transaction created a contract, store the creation address in the receipt.
	if msg.To == nil {
		// If the contract creation was a predeployed contract, we need to set the receipt's contract address to the
		// override address
		// Otherwise, we use the traditional method based on tx.origin and nonce
		if len(testChainConfig.ContractAddressOverrides) > 0 {
			initBytecodeHash := crypto.Keccak256Hash(msg.Data)
			if overrideAddr, ok := testChainConfig.ContractAddressOverrides[initBytecodeHash]; ok {
				receipt.ContractAddress = overrideAddr
			} else {
				receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
			}
		} else {
			receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
		}
	}

	// Set the receipt logs and create the bloom filter.
	receipt.Logs = statedb.GetLogs(tx.Hash(), blockNumber.Uint64(), blockHash)
	receipt.Bloom = gethtypes.CreateBloom(receipt)
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(statedb.TxIndex())
	return receipt, result, nil
}
