package chain

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core"
	"github.com/crytic/medusa-geth/core/tracing"
	gethTypes "github.com/crytic/medusa-geth/core/types"
	"github.com/crytic/medusa-geth/eth/tracers"
	"github.com/stretchr/testify/require"
)

// TestCallRouterReuse compares direct routing with the wrapper used when an
// additional tracer is supplied, including registrations made during a call.
func TestCallRouterReuse(t *testing.T) {
	t.Parallel()
	for _, reverted := range []bool{false, true} {
		t.Run(fmt.Sprintf("reverted=%v", reverted), func(t *testing.T) {
			t.Parallel()
			var transcripts [2][]string
			var results [2]*core.ExecutionResult
			var receipts [2]*gethTypes.Receipt
			for variant := range transcripts {
				c, senders := createChain(t)
				t.Cleanup(c.Close)
				root := common.HexToAddress("0x1234")
				child := common.HexToAddress("0x1235")
				// The root makes a nested STATICCALL and returns its result. The
				// child either returns one word or reverts with that same word.
				c.state.SetCode(root, []byte{0x60, 32, 0x60, 0, 0x60, 0, 0x60, 0, 0x61, 0x12, 0x35, 0x5a, 0xfa, 0x50, 0x60, 32, 0x60, 0, 0xf3})
				code := []byte{0x60, 7, 0x60, 0, 0x52, 0x60, 32, 0x60, 0, 0xf3}
				if reverted {
					code[len(code)-1] = 0xfd
				}
				c.state.SetCode(child, code)
				msg := &core.Message{From: senders[0], To: &root, GasLimit: 1000000,
					Value: new(big.Int), GasPrice: new(big.Int), GasFeeCap: new(big.Int), GasTipCap: new(big.Int)}
				record := func(format string, args ...any) {
					transcripts[variant] = append(transcripts[variant], fmt.Sprintf(format, args...))
				}
				late := &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{
					OnOpcode: func(pc uint64, op byte, _, _ uint64, _ tracing.OpContext, _ []byte, depth int, _ error) {
						record("late:%d:%d:%d", depth, pc, op)
					},
				}}}
				registered := false
				observer := &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{
					OnTxStart: func(_ *tracing.VMContext, tx *gethTypes.Transaction, from common.Address) {
						record("start:%s:%s", tx.Hash(), from)
					},
					OnEnter: func(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
						record("enter:%d:%d:%s:%s:%x:%d:%v", depth, typ, from, to, input, gas, value)
					},
					OnOpcode: func(pc uint64, op byte, gas, cost uint64, _ tracing.OpContext, data []byte, depth int, err error) {
						record("op:%d:%d:%d:%d:%d:%x:%v", depth, pc, op, gas, cost, data, err)
						// A tracer added during execution must be visible on the
						// next event through either route.
						if !registered {
							c.AddTracer(late, false, true)
							registered = true
						}
					},
					OnExit: func(depth int, output []byte, gas uint64, err error, reverted bool) {
						record("exit:%d:%x:%d:%v:%v", depth, output, gas, err, reverted)
					},
					OnTxEnd: func(receipt *gethTypes.Receipt, err error) {
						receipts[variant] = receipt
						record("end:%v", err)
					},
				}}}
				c.AddTracer(observer, false, true)
				var additional []*TestChainTracer
				if variant == 1 {
					// An empty additional tracer forces the original wrapper path.
					additional = []*TestChainTracer{{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{}}}}
				}
				result, err := c.CallContract(msg, nil, additional...)
				require.NoError(t, err)
				results[variant] = result
				require.True(t, registered)
				require.NotNil(t, receipts[variant])
			}
			require.Equal(t, transcripts[1], transcripts[0])
			require.Equal(t, results[1], results[0])
			require.Equal(t, receipts[1], receipts[0])
		})
	}
}

// TestCallRouterAdditionalTracers verifies callback order and that a tracer
// supplied for one call is not retained by the reusable router.
func TestCallRouterAdditionalTracers(t *testing.T) {
	t.Parallel()
	c, senders := createChain(t)
	t.Cleanup(c.Close)
	var events []string
	observer := func(name string) *TestChainTracer {
		return &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{
			OnTxEnd: func(*gethTypes.Receipt, error) { events = append(events, name) },
		}}}
	}
	c.AddTracer(observer("permanent"), false, true)
	to := common.HexToAddress("0x1234")
	msg := &core.Message{From: senders[0], To: &to, GasLimit: 1000000,
		Value: new(big.Int), GasPrice: new(big.Int), GasFeeCap: new(big.Int), GasTipCap: new(big.Int)}
	_, err := c.CallContract(msg, nil, observer("additional"))
	require.NoError(t, err)
	require.Equal(t, []string{"permanent", "additional"}, events)
	events = nil
	_, err = c.CallContract(msg, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"permanent"}, events)
}
