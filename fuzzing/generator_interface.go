package fuzzing

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type txGenerator interface {
	chooseMethod(worker *fuzzerWorker) *deployedMethod
	chooseSender(worker *fuzzerWorker) *fuzzerAccount
	generateAddress(worker *fuzzerWorker) common.Address
	generateArrayLength(worker *fuzzerWorker) int
	generateBool(worker *fuzzerWorker) bool
	generateBytes(worker *fuzzerWorker) []byte
	generateFixedBytes(worker *fuzzerWorker, length int) []byte
	generateString(worker *fuzzerWorker) string
	generateArbitraryUint(worker *fuzzerWorker, bitWidth int) *big.Int
	generateArbitraryInt(worker *fuzzerWorker, bitWidth int) *big.Int
	generateUint64(worker *fuzzerWorker) uint64
	generateInt64(worker *fuzzerWorker) int64
	generateUint32(worker *fuzzerWorker) uint32
	generateInt32(worker *fuzzerWorker) int32
	generateUint16(worker *fuzzerWorker) uint16
	generateInt16(worker *fuzzerWorker) int16
	generateUint8(worker *fuzzerWorker) uint8
	generateInt8(worker *fuzzerWorker) int8
}
