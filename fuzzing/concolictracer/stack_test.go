package concolictracer

import (
	"fmt"
	"github.com/mitchellh/go-z3"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConcolicStack(t *testing.T) {
	config := z3.NewConfig()
	ctx := z3.NewContext(config)
	solver := ctx.NewSolver()
	defer config.Close()
	defer ctx.Close()
	defer solver.Close()

	stack := newConcolicStack()
	five := ctx.Const(ctx.Symbol("willBeFive"), ctx.IntSort())

	concVar := &concolicVariable{
		variable: five,
	}

	stack.pushVariable(concVar)
	stack.dupeN(1)

	duped := stack.getVariable(1)
	orig := stack.getVariable(2)

	doubledDuped := duped.variable.Add(five)

	ten := ctx.Int(10, ctx.IntSort())
	solver.Assert(doubledDuped.Eq(ten))

	five_conc := ctx.Int(5, ctx.IntSort())
	solver.Assert(orig.variable.Eq(five_conc))
	solver.Assert(doubledDuped.Eq(five_conc).Not())

	v := solver.Check()

	assert.True(t, v == z3.True)

	m := solver.Model()
	assignments := m.Assignments()
	m.Close()
	for k, v := range assignments {
		fmt.Printf("%s = %s\n", k, v)
	}

}
