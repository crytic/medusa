package fuzzer

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type txGenerator interface {
	chooseMethod(worker *fuzzerWorker) *deployedMethod
	chooseSender(worker *fuzzerWorker) *fuzzerAccount
	generateAddress(worker *fuzzerWorker) common.Address
	generateUint(worker *fuzzerWorker) *big.Int
}
