package chain

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core/tracing"
	"github.com/crytic/medusa-geth/core/vm"
	"github.com/crytic/medusa-geth/eth/tracers"
	"github.com/stretchr/testify/require"
)

// TestOpcodeRouterMutableHooks verifies that registration does not snapshot a
// public tracer's callback or hook container, including an initially nil hook.
func TestOpcodeRouterMutableHooks(t *testing.T) {
	t.Parallel()
	for _, route := range []string{"direct", "native", "nested"} {
		t.Run(route, func(t *testing.T) {
			t.Parallel()
			router := NewTestChainTracerRouter()
			observer := &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{}}}
			router.AddTracer(observer)
			dispatch := router.OnOpcode
			switch route {
			case "native":
				dispatch = router.NativeTracer().OnOpcode
			case "nested":
				parent := NewTestChainTracerRouter()
				parent.AddTracer(router.NativeTracer())
				dispatch = parent.NativeTracer().OnOpcode
			}
			var events []string
			hook := func(name string) tracing.OpcodeHook {
				return func(_ uint64, op byte, _, _ uint64, _ tracing.OpContext, _ []byte, _ int, _ error) {
					events = append(events, fmt.Sprintf("%s:%d", name, op))
				}
			}
			check := func(name string) {
				t.Helper()
				events = nil
				var want []string
				for op := range 256 {
					dispatch(0, byte(op), 0, 0, nil, nil, 0, nil)
					if name != "" {
						want = append(want, fmt.Sprintf("%s:%d", name, op))
					}
				}
				require.Equal(t, want, events)
			}
			check("")
			observer.OnOpcode = hook("enabled")
			check("enabled")
			observer.OnOpcode = hook("replaced")
			check("replaced")
			observer.Hooks = &tracing.Hooks{OnOpcode: hook("hooks")}
			check("hooks")
			observer.Tracer = &tracers.Tracer{Hooks: &tracing.Hooks{OnOpcode: hook("tracer")}}
			check("tracer")
			observer.OnOpcode = nil
			check("")
		})
	}
}

// TestOpcodeRouterReentrantUpdates checks the original range semantics: hook
// changes affect later observers immediately, while additions join the next
// event, including a reentrant event before the current dispatch finishes.
func TestOpcodeRouterReentrantUpdates(t *testing.T) {
	t.Parallel()
	router := NewTestChainTracerRouter()
	parent := NewTestChainTracerRouter()
	parent.AddTracer(router.NativeTracer())
	var events []string
	observer := func(name string) *TestChainTracer {
		return &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{
			OnOpcode: func(pc uint64, _ byte, _, _ uint64, _ tracing.OpContext, _ []byte, _ int, _ error) {
				events = append(events, fmt.Sprintf("%s:%d", name, pc))
			},
		}}}
	}
	first, second, third, late := observer("first"), observer("second"), observer("third"), observer("late")
	second.OnOpcode = nil
	first.OnOpcode = func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, data []byte, depth int, err error) {
		events = append(events, fmt.Sprintf("first:%d", pc))
		if pc == 0 {
			second.OnOpcode = observer("updated").OnOpcode
			third.OnOpcode = nil
			router.AddTracer(late)
			parent.OnOpcode(1, op, gas, cost, scope, data, depth, err)
		}
	}
	router.AddTracers(first, second, third)
	parent.OnOpcode(0, byte(vm.ADD), 0, 0, nil, nil, 0, nil)
	require.Equal(t, []string{"first:0", "first:1", "updated:1", "late:1", "updated:0"}, events)
	events = nil
	parent.OnOpcode(2, byte(vm.ADD), 0, 0, nil, nil, 0, nil)
	require.Equal(t, []string{"first:2", "updated:2", "late:2"}, events)
}

// TestOpcodeRouterSubscriptions checks the private fixed subscription without
// restricting public observers or changing non-opcode event delivery.
func TestOpcodeRouterSubscriptions(t *testing.T) {
	t.Parallel()
	router := NewTestChainTracerRouter()
	var events []string
	observer := func(name string) *TestChainTracer {
		return &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{
			OnOpcode: func(_ uint64, _ byte, _, _ uint64, _ tracing.OpContext, _ []byte, _ int, _ error) {
				events = append(events, name)
			},
			OnEnter: func(_ int, _ byte, _, _ common.Address, _ []byte, _ uint64, _ *big.Int) {
				events = append(events, name)
			},
		}}}
	}
	first, filtered, last := observer("first"), observer("filtered"), observer("last")
	router.AddTracer(first)
	router.addTracerForOpcodes(filtered, vm.SELFDESTRUCT, vm.SELFDESTRUCT)
	router.AddTracer(last)
	require.Equal(t, []*TestChainTracer{first, filtered, last}, router.Tracers())
	// The per-frame dispatcher must honor the same subscriptions.
	frame := router.NativeTracer().OnOpcodeForScope(nil)
	for op := range 256 {
		want := []string{"first", "last"}
		if vm.OpCode(op) == vm.SELFDESTRUCT {
			want = []string{"first", "filtered", "last"}
		}
		events = nil
		router.NativeTracer().OnOpcode(0, byte(op), 0, 0, nil, nil, 0, nil)
		require.Equal(t, want, events, "opcode %d", op)
		events = nil
		frame(0, byte(op), 0, 0, nil, nil, 0, nil)
		require.Equal(t, want, events, "scoped opcode %d", op)
	}
	events = nil
	router.OnEnter(0, 0, common.Address{}, common.Address{}, nil, 0, nil)
	require.Equal(t, []string{"first", "filtered", "last"}, events)
}
