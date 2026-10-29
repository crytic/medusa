package chain

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core"
	"github.com/crytic/medusa-geth/core/tracing"
	"github.com/crytic/medusa-geth/eth/tracers"
	"github.com/stretchr/testify/require"
)

// TestCallRouterLateObservers compares the sole internal tracer with generic
// routing when a call-specific observer registers a permanent observer during
// execution. The existing wrapper must see the updated route on its next event.
func TestCallRouterLateObservers(t *testing.T) {
	t.Parallel()
	for _, registerOn := range []string{"enter", "opcode"} {
		t.Run(registerOn, func(t *testing.T) {
			t.Parallel()
			var transcripts [2][]string
			var results [2][]*core.ExecutionResult
			for variant := range transcripts {
				c, senders := createChain(t)
				t.Cleanup(c.Close)
				root := common.HexToAddress("0x1234")
				child := common.HexToAddress("0x1235")
				c.state.SetCode(root, []byte{0x60, 32, 0x60, 0, 0x60, 0, 0x60, 0, 0x61, 0x12, 0x35, 0x5a, 0xfa, 0x50, 0x60, 32, 0x60, 0, 0xf3})
				c.state.SetCode(child, []byte{0x60, 7, 0x60, 0, 0x52, 0x60, 32, 0x60, 0, 0xf3})
				if variant == 0 {
					// An empty permanent observer selects generic routing without
					// adding any observable callbacks.
					c.AddTracer(&TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{}}}, false, true)
				}
				hook := func(name string) tracing.OpcodeHook {
					return func(pc uint64, op byte, gas, cost uint64, _ tracing.OpContext, data []byte, depth int, err error) {
						transcripts[variant] = append(transcripts[variant], fmt.Sprintf("%s:%d:%d:%d:%d:%d:%x:%v", name, depth, pc, op, gas, cost, data, err))
					}
				}
				observer := &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{}}}
				registered := false
				register := func() {
					if !registered {
						c.AddTracer(observer, false, true)
						observer.OnOpcode = hook("late")
						registered = true
					}
				}
				additional := &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{}}}
				if registerOn == "enter" {
					additional.OnEnter = func(_ int, _ byte, _, _ common.Address, _ []byte, _ uint64, _ *big.Int) { register() }
				} else {
					additional.OnOpcode = func(_ uint64, _ byte, _, _ uint64, _ tracing.OpContext, _ []byte, _ int, _ error) { register() }
				}
				msg := &core.Message{From: senders[0], To: &root, GasLimit: 1000000,
					Value: new(big.Int), GasPrice: new(big.Int), GasFeeCap: new(big.Int), GasTipCap: new(big.Int)}
				result, err := c.CallContract(msg, nil, additional)
				require.NoError(t, err)
				require.True(t, registered)
				require.NotEmpty(t, transcripts[variant])
				results[variant] = append(results[variant], result)
				observer.Hooks = &tracing.Hooks{OnOpcode: hook("replacement")}
				result, err = c.CallContract(msg, nil)
				require.NoError(t, err)
				results[variant] = append(results[variant], result)
				count := len(transcripts[variant])
				observer.OnOpcode = nil
				result, err = c.CallContract(msg, nil)
				require.NoError(t, err)
				results[variant] = append(results[variant], result)
				require.Len(t, transcripts[variant], count)
			}
			require.Equal(t, transcripts[0], transcripts[1])
			require.Equal(t, results[0], results[1])
		})
	}
}
