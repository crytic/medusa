package chain

import (
	"math/big"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core"
	"github.com/crytic/medusa-geth/core/types"
	"github.com/crytic/medusa-geth/core/vm"
)

// newTestChainBlockContext obtains a new vm.BlockContext that is tailored to provide data from a TestChain.
func newTestChainBlockContext(testChain *TestChain, header *types.Header) vm.BlockContext {
	return vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash: func(n uint64) common.Hash {
			// Obtain our block hash from this number. If an error occurs, we ignore it and simply return an empty dummy
			// hash.
			hash, _ := testChain.BlockHashFromNumber(n)
			return hash
		},
		Coinbase:    header.Coinbase,
		BlockNumber: new(big.Int).Set(header.Number),
		Time:        header.Time,
		Difficulty:  new(big.Int).Set(header.Difficulty),
		BaseFee:     new(big.Int).Set(testChain.Head().Header.BaseFee),
		GasLimit:    header.GasLimit,
		Random:      &header.MixDigest,
	}
}
