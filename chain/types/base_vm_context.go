package types

import "math/big"

type BaseVMContext struct {
	Number *big.Int
	Time   uint64
}

func NewBaseVMContext() *BaseVMContext {
	return &BaseVMContext{
		Number: big.NewInt(0),
		Time:   0,
	}
}
