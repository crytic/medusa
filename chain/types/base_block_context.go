package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// BaseBlockContext stores block-level information (e.g. block.number or block.timestamp) when the block is first
// created. We need to store these values because cheatcodes like warp or roll will directly modify the block header.
// We use these values during the cloning process to ensure that execution semantics are maintained while still
// allowing the cheatcodes to function as expected.
type BaseBlockContext struct {
	// number represents the block number of the block when it was first created.
	number *big.Int
	// time represents the timestamp of the block when it was first created.
	time uint64
	// baseFee represents the base fee of the block when it was first created.
	baseFee *big.Int
	// coinbase represents the coinbase of the block when it was first created.
	coinbase common.Address
	// difficulty represents the difficulty of the block when it was first created.
	difficulty *big.Int
	// random represents the random value (used for PREVRANDAO) of the block when it was first created.
	random *common.Hash
}

// NewBaseBlockContext returns a new BaseBlockContext with the provided parameters.
func NewBaseBlockContext(number *big.Int, time uint64, baseFee *big.Int, coinbase common.Address, difficulty *big.Int, random *common.Hash) *BaseBlockContext {
	return &BaseBlockContext{
		number:     number,
		time:       time,
		baseFee:    baseFee,
		coinbase:   coinbase,
		random:     random,
		difficulty: difficulty,
	}
}
