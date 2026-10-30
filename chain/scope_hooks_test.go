package chain

import (
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core/tracing"
	"github.com/crytic/medusa-geth/core/vm"
	"github.com/crytic/medusa-geth/eth/tracers"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// TestScopeRouterLiveCallbacks verifies that ordinary callbacks and late
// registrations retain their existing behavior inside an already selected frame.
func TestScopeRouterLiveCallbacks(t *testing.T) {
	router := NewTestChainTracerRouter()
	hook := router.OnOpcodeForScope(nil)
	require.NotNil(t, hook)
	calls := []string{}
	late := &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{OnOpcode: func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) {
		calls = append(calls, "late")
	}}}}
	first := &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{}}}
	first.OnOpcode = func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) {
		calls = append(calls, "first")
		router.AddTracer(late)
		first.OnOpcode = func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) {
			calls = append(calls, "replacement")
		}
	}
	router.AddTracer(first)
	hook(0, 0, 0, 0, nil, nil, 0, nil)
	require.Equal(t, []string{"first"}, calls)
	hook(0, 0, 0, 0, nil, nil, 0, nil)
	require.Equal(t, []string{"first", "replacement", "late"}, calls)
	first.OnOpcode = nil
	hook(0, 0, 0, 0, nil, nil, 0, nil)
	require.Equal(t, []string{"first", "replacement", "late", "late"}, calls)
}

// TestScopeRouterNestedFrames checks that frame selections do not overwrite
// their parent's callbacks and that an opted-in nil selection stays silent.
func TestScopeRouterNestedFrames(t *testing.T) {
	router := NewTestChainTracerRouter()
	selections, calls := 0, 0
	tracer := &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{
		OnOpcode: func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) {
			t.Error("unexpected fallback")
		},
		OnOpcodeForScope: func(tracing.OpContext) tracing.OpcodeHook {
			selections++
			if selections == 2 {
				return nil
			}
			return func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) { calls++ }
		},
	}}}
	router.AddTracer(tracer)
	parent := router.OnOpcodeForScope(nil)
	child := router.OnOpcodeForScope(nil)
	parent(0, 0, 0, 0, nil, nil, 0, nil)
	child(0, 0, 0, 0, nil, nil, 0, nil)
	parent(0, 0, 0, 0, nil, nil, 0, nil)
	require.Equal(t, 2, selections)
	require.Equal(t, 2, calls)
}

// TestCheatScopeEntryHooks verifies that the next-child hooks still run on the
// first opcode, after the child scope is available, and are consumed once.
func TestCheatScopeEntryHooks(t *testing.T) {
	tracer := newCheatCodeTracer()
	tracer.OnTxStart(nil, nil, common.Address{})
	tracer.OnEnter(0, byte(vm.CALL), common.Address{}, common.Address{}, nil, 100, nil)
	parent := &vm.ScopeContext{Contract: vm.NewContract(common.Address{}, common.Address{}, new(uint256.Int), 100, nil)}
	require.Nil(t, tracer.OnOpcodeForScope(parent))
	require.Same(t, parent, tracer.CurrentCallFrame().vmScope)
	calls := 0
	tracer.CurrentCallFrame().onNextFrameEnterHooks = append(tracer.CurrentCallFrame().onNextFrameEnterHooks, func() { calls++ })
	tracer.OnEnter(1, byte(vm.CALL), common.Address{}, common.Address{}, nil, 100, nil)
	child := &vm.ScopeContext{Contract: vm.NewContract(common.Address{}, common.Address{}, new(uint256.Int), 100, nil)}
	hook := tracer.OnOpcodeForScope(child)
	require.NotNil(t, hook)
	require.Zero(t, calls)
	hook(0, byte(vm.STOP), 100, 0, child, nil, 1, nil)
	hook(0, byte(vm.STOP), 100, 0, child, nil, 1, nil)
	require.Equal(t, 1, calls)
	require.Same(t, child, tracer.CurrentCallFrame().vmScope)
}

// TestScopeRouterLateSelection verifies that opt-in subscribers registered
// after a frame starts are selected once and receive subsequent events.
func TestScopeRouterLateSelection(t *testing.T) {
	router := NewTestChainTracerRouter()
	hook := router.OnOpcodeForScope(nil)
	selected, calls := 0, 0
	router.AddTracer(&TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{
		OnOpcodeForScope: func(tracing.OpContext) tracing.OpcodeHook {
			selected++
			return func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) { calls++ }
		},
	}}})
	hook(0, 0, 0, 0, nil, nil, 0, nil)
	hook(0, 0, 0, 0, nil, nil, 0, nil)
	require.Equal(t, 1, selected)
	require.Equal(t, 2, calls)
}

// TestScopeRouterDirectFrames checks the direct path for frames with fixed
// hooks and its fallback for fixed subscriptions and late registrations.
func TestScopeRouterDirectFrames(t *testing.T) {
	router := NewTestChainTracerRouter()
	var calls []string
	record := func(name string) tracing.OpcodeHook {
		return func(_ uint64, op byte, _, _ uint64, _ tracing.OpContext, _ []byte, _ int, _ error) {
			calls = append(calls, name+":"+vm.OpCode(op).String())
		}
	}
	scoped := func(name string, hook tracing.OpcodeHook) *TestChainTracer {
		return &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{
			OnOpcode:         record(name + "-unused"),
			OnOpcodeForScope: func(tracing.OpContext) tracing.OpcodeHook { return hook },
		}}}
	}
	router.AddTracer(scoped("silent", nil))
	router.addTracerForOpcodes(&TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{
		OnOpcode: record("fixed"),
	}}}, vm.SELFDESTRUCT)
	router.AddTracer(scoped("scoped", record("scoped")))
	hook := router.OnOpcodeForScope(nil)
	hook(0, byte(vm.ADD), 0, 0, nil, nil, 0, nil)
	hook(0, byte(vm.SELFDESTRUCT), 0, 0, nil, nil, 0, nil)
	require.Equal(t, []string{"scoped:ADD", "fixed:SELFDESTRUCT", "scoped:SELFDESTRUCT"}, calls)

	// A second selected hook needs the ordered dispatcher.
	router.AddTracer(scoped("second", record("second")))
	calls = nil
	hook(0, byte(vm.ADD), 0, 0, nil, nil, 0, nil)
	require.Equal(t, []string{"scoped:ADD", "second:ADD"}, calls)
	calls = nil
	router.OnOpcodeForScope(nil)(0, byte(vm.ADD), 0, 0, nil, nil, 0, nil)
	require.Equal(t, []string{"scoped:ADD", "second:ADD"}, calls)

	// An ordinary public tracer keeps its live hook.
	live := &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{}}}
	router.AddTracer(live)
	hook = router.OnOpcodeForScope(nil)
	live.OnOpcode = record("live")
	calls = nil
	hook(0, byte(vm.ADD), 0, 0, nil, nil, 0, nil)
	require.Equal(t, []string{"scoped:ADD", "second:ADD", "live:ADD"}, calls)
}
