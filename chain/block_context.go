package chain

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"
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
		Time:        new(big.Int).SetUint64(header.Time),
		Difficulty:  new(big.Int).Set(header.Difficulty),
		BaseFee:     new(big.Int).Set(testChain.Head().Header().BaseFee),
		GasLimit:    header.GasLimit,
		Random:      &header.MixDigest,
	}
}
