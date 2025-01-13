package concolictracer

import "github.com/ethereum/go-ethereum/common"

type concolicCallFrame struct {
	Address  common.Address
	stack    *concolicStack
	memory   map[common.Hash]*concolicVariable
	tstorage map[common.Hash]common.Hash
}

func newConcolicCallFrame(address common.Address) *concolicCallFrame {
	return &concolicCallFrame{
		Address:  address,
		stack:    newConcolicStack(),
		memory:   make(map[common.Hash]*concolicVariable),
		tstorage: make(map[common.Hash]common.Hash),
	}
}
