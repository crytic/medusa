package types

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sync"
)

// Corpus holds a list of corpus transaction sequences
type Corpus struct {
	// List of corpus transaction sequences
	TransactionSequences [][]*MetaTx
	// Mutex allows for concurrent reads but syncs on writes
	Mutex sync.RWMutex
}

// MetaTx is the core components of a transaction and is used for storing and replaying transactions in the Corpus
type MetaTx struct {
	// Nonce is the nonce
	Nonce uint64
	// Src is the source of the transaction
	// TODO: Can this be a common.Address?
	Src *common.Address
	// Dst is the destination of the transaction
	Dst *common.Address
	// Gas is gas
	Gas uint64
	// GasPrice is gas price
	GasPrice *big.Int
	// Data is data
	Data []byte
	// Value is value
	Value *big.Int
}

//NewMetaTx creates a default MetaTx object
func NewMetaTx() *MetaTx {
	return &MetaTx{
		Nonce:    0,
		Src:      &common.Address{0},
		Dst:      &common.Address{0},
		Gas:      0,
		GasPrice: big.NewInt(0),
		Data:     []byte{0},
		Value:    big.NewInt(0),
	}
}
