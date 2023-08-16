package utils

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
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
	})
}
