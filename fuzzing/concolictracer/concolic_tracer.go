package concolictracer

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/mitchellh/go-z3"
)

type concolicTx struct {
	calldata map[int]*z3.AST
	z3ctx    *z3.Context
	solver   *z3.Solver

	storage map[common.Hash]*concolicVariable

	callStack []*concolicCallFrame

	// used for transient storage retrieval
	inactiveFrames map[common.Address]*concolicCallFrame

	cleanup func()
}

func newConcolicTx() *concolicTx {
	config := z3.NewConfig()
	ctx := z3.NewContext(config)
	solver := ctx.NewSolver()

	cleanup := func() {
		config.Close()
		ctx.Close()
		solver.Close()
	}

	return &concolicTx{
		calldata:       make(map[int]*z3.AST),
		z3ctx:          ctx,
		solver:         solver,
		storage:        make(map[common.Hash]*concolicVariable),
		callStack:      make([]*concolicCallFrame, 0),
		inactiveFrames: make(map[common.Address]*concolicCallFrame),
		cleanup:        cleanup,
	}
}

func test() {
	tx := newConcolicTx()

	// one symbol for each calldata word
	//x := tx.z3ctx.Const(tx.z3ctx.Symbol("x"), tx.z3ctx.IntSort())
	//y := tx.z3ctx.Const(tx.z3ctx.Symbol("y"), tx.z3ctx.IntSort())
	//z := tx.z3ctx.Const(tx.z3ctx.Symbol("z"), tx.z3ctx.IntSort())
	//zero := tx.z3ctx.Int(0, tx.z3ctx.IntSort()) // To save repeats
	//za := x.Add(y)

	// for shr/shl, multiply or divide integer by two
}
