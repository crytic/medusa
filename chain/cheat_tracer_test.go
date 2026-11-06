package chain

import (
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core/vm"
	"github.com/stretchr/testify/require"
)

// TestCheatTracerLiveScope checks that a frame retains a live scope and that
// hooks queued by another hook still run at the next opcode in that frame.
func TestCheatTracerLiveScope(t *testing.T) {
	t.Parallel()
	tracer := newCheatCodeTracer()
	tracer.OnTxStart(nil, nil, common.Address{})
	tracer.OnEnter(0, byte(vm.CALL), common.Address{}, common.Address{}, nil, 100000, nil)
	parent := tracer.CurrentCallFrame()
	parentScope := &vm.ScopeContext{Memory: vm.NewMemory()}
	tracer.OnOpcode(0, byte(vm.STOP), 100000, 0, parentScope, nil, 0, nil)
	require.Same(t, parentScope, parent.vmScope)

	// Enter hooks need the child's scope, so they execute at its first opcode.
	childScope := &vm.ScopeContext{Memory: vm.NewMemory()}
	var events []int
	parent.onNextFrameEnterHooks.Push(func() {
		require.Same(t, childScope, tracer.CurrentCallFrame().vmScope)
		events = append(events, 1)
		parent.onNextFrameEnterHooks.Push(func() { events = append(events, 2) })
	})
	tracer.OnEnter(1, byte(vm.CALL), common.Address{}, common.Address{}, nil, 90000, nil)
	child := tracer.CurrentCallFrame()
	tracer.OnOpcode(0, byte(vm.PUSH1), 90000, 3, childScope, nil, 1, nil)
	require.Equal(t, []int{1}, events)
	childScope.Memory.Resize(32)
	tracer.OnOpcode(2, byte(vm.STOP), 89997, 0, childScope, nil, 1, nil)
	require.Equal(t, []int{1, 2}, events)
	require.Len(t, child.vmScope.MemoryData(), 32)
	tracer.OnOpcode(3, byte(vm.STOP), 89997, 0, childScope, nil, 1, nil)
	require.Equal(t, []int{1, 2}, events)

	// Returning to the parent and entering a sibling must retain separate scopes.
	tracer.OnExit(1, nil, 3, nil, false)
	require.Same(t, parentScope, tracer.CurrentCallFrame().vmScope)
	tracer.OnEnter(1, byte(vm.CALL), common.Address{}, common.Address{}, nil, 80000, nil)
	require.Nil(t, tracer.CurrentCallFrame().vmScope)
	siblingScope := &vm.ScopeContext{Memory: vm.NewMemory()}
	tracer.OnOpcode(0, byte(vm.STOP), 80000, 0, siblingScope, nil, 1, nil)
	require.Same(t, siblingScope, tracer.CurrentCallFrame().vmScope)
	require.Same(t, childScope, child.vmScope)
}
