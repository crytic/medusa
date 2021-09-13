package fuzzing

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// txGenerator represents an interface for a provider used to generate transaction fields and call arguments for use
// in fuzzing campaigns.
type txGenerator interface {
	// chooseMethod selects a random state-changing deployed contract method to target with a transaction.
	chooseMethod(worker *fuzzerWorker) *deployedMethod
	// chooseSender selects a random account address to send the transaction from.
	chooseSender(worker *fuzzerWorker) *fuzzerAccount
	// generateAddress generates/selects an address to use when populating transaction fields.
	generateAddress(worker *fuzzerWorker) common.Address
	// generateArrayLength generates/selects an array length to use when populating transaction fields.
	generateArrayLength(worker *fuzzerWorker) int
	// generateBool generates/selects a bool to use when populating transaction fields.
	generateBool(worker *fuzzerWorker) bool
	// generateBytes generates/selects a dynamic-sized byte array to use when populating transaction fields.
	generateBytes(worker *fuzzerWorker) []byte
	// generateFixedBytes generates/selects a fixed-sized byte array to use when populating transaction fields.
	generateFixedBytes(worker *fuzzerWorker, length int) []byte
	// generateString generates/selects a dynamic-sized string to use when populating transaction fields.
	generateString(worker *fuzzerWorker) string
	// generateInteger generates/selects an integer to use when populating transaction fields.
	generateInteger(worker *fuzzerWorker, signed bool, bitLength int) *big.Int
}
