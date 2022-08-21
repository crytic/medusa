package types

import (
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/sha3"
	"strconv"
	"strings"
	"sync"
)

// Corpus holds a list of transaction sequences
type Corpus struct {
	// TransactionSequences is a mapping between the TransactionSequenceHash of a CallMessage sequence and the CallMessage sequence
	TransactionSequences map[string][]*CallMessage
	// Mutex allows for concurrent reads but syncs on writes
	Mutex sync.Mutex
	// WriteIndex is an index in the TransactionSequences list that points to the next object to be written to disk
	WriteIndex uint64
}

// NewCorpus initializes a new Corpus object for the Fuzzer
func NewCorpus() *Corpus {
	return &Corpus{
		TransactionSequences: map[string][]*CallMessage{},
		WriteIndex:           0,
	}
}

// TransactionSequenceHash takes in an array of CallMessage and hashes it
func (c *Corpus) TransactionSequenceHash(msgSequence []*CallMessage) string {
	// Calculate a hash which is unique per msg sequence
	var msgSequenceString string
	for _, msg := range msgSequence {
		msgSequenceString = msgSequenceString + strings.Join([]string{msg.From().String(), msg.To().String(),
			msg.Value().String(), strconv.FormatUint(msg.Nonce(), 10), fmt.Sprintf("%s", msg.Data()),
			strconv.FormatUint(msg.Gas(), 10), msg.GasFeeCap().String(), msg.GasTipCap().String(),
			msg.GasPrice().String()}, ",")
	}
	hash := sha3.NewLegacyKeccak256().Sum([]byte(msgSequenceString))
	return hex.EncodeToString(hash)
}

// MetaTx is a dummy object that we can use in the future to store a CallMessage plus extra stuff
type MetaTx struct{}

//NewMetaTx creates a default MetaTx object
func NewMetaTx() *MetaTx {
	return &MetaTx{}
}
