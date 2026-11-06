package chain

import (
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core/tracing"
	gethTypes "github.com/crytic/medusa-geth/core/types"
	"github.com/crytic/medusa-geth/eth/tracers"
	"github.com/crytic/medusa/utils"
	"github.com/stretchr/testify/require"
)

type receiptTestPrecompile struct{ run func() }

func (*receiptTestPrecompile) RequiredGas([]byte) uint64 { return 13 }
func (p *receiptTestPrecompile) Run([]byte) ([]byte, error) {
	p.run()
	return make([]byte, 32), nil
}

func TestCallReceiptObservers(t *testing.T) {
	for _, mode := range []string{"permanent", "additional", "late"} {
		t.Run(mode, func(t *testing.T) {
			a, _, msg := memoTestPair(t, memoNested)
			wantTx := utils.MessageToTransaction(msg)
			var receipt *gethTypes.Receipt
			starts := 0
			observer := &TestChainTracer{Tracer: &tracers.Tracer{Hooks: &tracing.Hooks{
				OnTxStart: func(_ *tracing.VMContext, tx *gethTypes.Transaction, from common.Address) {
					starts++
					require.Equal(t, wantTx.Hash(), tx.Hash())
					require.Equal(t, msg.From, from)
				},
				OnTxEnd: func(r *gethTypes.Receipt, err error) {
					require.NoError(t, err)
					receipt = r
				},
			}}}
			var extra []*TestChainTracer
			switch mode {
			case "permanent":
				a.AddTracer(observer, false, true)
			case "additional":
				extra = append(extra, observer)
			case "late":
				a.vmConfigExtensions.AdditionalPrecompiles[memoChild] = &receiptTestPrecompile{run: func() {
					a.AddTracer(observer, false, true)
					// The receipt must describe the transaction as it was at entry,
					// even if an extension changes the caller-owned message later.
					msg.Data = []byte{1, 2, 3}
				}}
			}
			result, err := a.CallContract(msg, nil, extra...)
			require.NoError(t, err)
			require.NotNil(t, receipt)
			require.Equal(t, wantTx.Hash(), receipt.TxHash)
			require.Equal(t, result.UsedGas, receipt.GasUsed)
			require.Equal(t, uint64(gethTypes.ReceiptStatusSuccessful), receipt.Status)
			if mode == "late" {
				require.Zero(t, starts)
			} else {
				require.Equal(t, 1, starts)
			}
		})
	}
}
