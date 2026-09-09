package chain

import (
	"context"
	"math/big"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core"
	"github.com/crytic/medusa-geth/core/tracing"
	gethTypes "github.com/crytic/medusa-geth/core/types"
	"github.com/crytic/medusa-geth/eth/tracers"
	"github.com/stretchr/testify/require"
)

func newCallBenchmarkChain(tb testing.TB) (*TestChain, *core.Message) {
	tb.Helper()
	address := common.Address{1}
	// Store 1 in slot 0, then return. CallContract must roll the storage write back.
	code := []byte{0x60, 0x01, 0x60, 0x00, 0x55, 0x60, 0x00, 0x60, 0x00, 0xf3}
	alloc := gethTypes.GenesisAlloc{address: {Code: code, Balance: new(big.Int)}}
	chain, err := NewTestChain(context.Background(), alloc, nil)
	require.NoError(tb, err)
	tb.Cleanup(chain.Close)
	msg := &core.Message{
		From: common.Address{2}, To: &address, GasLimit: 100000,
		Value: new(big.Int), GasPrice: new(big.Int),
		GasFeeCap: new(big.Int), GasTipCap: new(big.Int),
		SkipNonceChecks: true, SkipFromEOACheck: true,
	}
	return chain, msg
}

func TestCallContractTracerRouting(t *testing.T) {
	t.Parallel()
	chain, msg := newCallBenchmarkChain(t)
	var events []string
	newTracer := func(name string) *TestChainTracer {
		return &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{
			OnTxStart: func(*tracing.VMContext, *gethTypes.Transaction, common.Address) {
				events = append(events, name+" start")
			},
			OnOpcode: func(uint64, byte, uint64, uint64, tracing.OpContext, []byte, int, error) {
				events = append(events, name+" opcode")
			},
			OnTxEnd: func(*gethTypes.Receipt, error) {
				events = append(events, name+" end")
			},
		}}}
	}
	chain.AddTracer(newTracer("base"), false, true)
	extra := newTracer("extra")
	for _, withExtra := range []bool{false, true, false} {
		events = nil
		var additional []*TestChainTracer
		if withExtra {
			additional = append(additional, extra)
		}
		result, err := chain.CallContract(msg, nil, additional...)
		require.NoError(t, err)
		require.False(t, result.Failed())
		expected := []string{"base start"}
		if withExtra {
			expected = append(expected, "extra start")
		}
		for i := 0; i < 6; i++ {
			expected = append(expected, "base opcode")
			if withExtra {
				expected = append(expected, "extra opcode")
			}
		}
		expected = append(expected, "base end")
		if withExtra {
			expected = append(expected, "extra end")
		}
		require.Equal(t, expected, events)
		require.Equal(t, common.Hash{}, chain.State().GetState(*msg.To, common.Hash{}))
	}
}

// BenchmarkChainCall measures provider-style calls with the default chain tracers enabled.
func BenchmarkChainCall(b *testing.B) {
	chain, msg := newCallBenchmarkChain(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := chain.CallContract(msg, nil)
		if err != nil || result.Failed() {
			b.Fatalf("contract call failed: result=%v err=%v", result, err)
		}
	}
}
