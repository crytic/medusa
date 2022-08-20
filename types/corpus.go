package types

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sync"
)

// Corpus holds a list of corpus transaction sequences
type Corpus struct {
	// TransactionSequences is a list of meta transaction sequences
	TransactionSequences [][]MetaTx
	// Mutex allows for concurrent reads but syncs on writes
	Mutex sync.RWMutex
	// WriteIndex is an index in the TransactionSequences list that points to the next object to be written to disk
	WriteIndex int
}

// NewCorpus initializes a new Corpus object for the Fuzzer
func NewCorpus() *Corpus {
	return &Corpus{
		TransactionSequences: [][]MetaTx{},
	}
}

// MetaTx is the core components of a transaction and is used for storing and replaying transactions in the Corpus
type MetaTx struct {
	// Nonce is the nonce field in the original transaction
	Nonce uint64
	// Value is value field in the original transaction
	Value *big.Int
	// Src is the source of the original transaction
	// TODO: Can this be a common.Address?
	Src *common.Address
	// Dst is the destination of the original transaction
	Dst *common.Address
	// Gas is the gas field in the original transaction
	Gas uint64
	// GasPrice is the gas price field in the original transaction
	GasPrice *big.Int
	// Data is the data field in the original transaction
	Data []byte
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
