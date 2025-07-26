package types

import (
	"math/big"

	"github.com/crytic/medusa-geth/common"
)

// BaseBlockContext stores block-level information (e.g. block.number or block.timestamp) when the block is first
// created. We need to store these values because cheatcodes like warp or roll will directly modify the block header.
// We use these values during the cloning process to ensure that execution semantics are maintained while still
// allowing the cheatcodes to function as expected. We could expand this struct to hold additional values
// (e.g. difficulty) but we will err to add values only as necessary.
type BaseBlockContext struct {
	// Number represents the block number of the block when it was first created.
	Number *big.Int
	// Time represents the timestamp of the block when it was first created.
	Time uint64
	// BaseFee represents the base fee of the block when it was first created.
	BaseFee *big.Int
	// Coinbase represents the coinbase of the block when it was first created.
	Coinbase common.Address
}

// NewBaseBlockContext returns a new BaseBlockContext with the provided parameters.
func NewBaseBlockContext(number uint64, time uint64, baseFee *big.Int, coinbase common.Address) *BaseBlockContext {
	return &BaseBlockContext{
		Number:   new(big.Int).SetUint64(number),
		Time:     time,
		BaseFee:  new(big.Int).Set(baseFee),
		Coinbase: coinbase,
	}
}
