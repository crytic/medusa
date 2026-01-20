package utils

import (
	"github.com/crytic/medusa-geth/core"
	"github.com/crytic/medusa-geth/core/types"
)

// MessageToTransaction derives a types.Transaction from a types.Message.
func MessageToTransaction(msg *core.Message) *types.Transaction {
	return types.NewTx(&types.LegacyTx{
		Nonce:    msg.Nonce,
		GasPrice: msg.GasPrice,
		Gas:      msg.GasLimit,
		To:       msg.To,
		Value:    msg.Value,
		Data:     msg.Data,
		// HACK: to avoid transactions from different senders hashing to
		// the same Hash() / receipt, we stuff the From address on one of
		// the signature parameters
		S: msg.From.Big(),
	})
}
