package corpus

import (
	"testing"

	"github.com/crytic/medusa-geth/common"
	chainTypes "github.com/crytic/medusa/chain/types"
	"github.com/crytic/medusa/fuzzing/calls"
	"github.com/crytic/medusa/fuzzing/coverage"
	"github.com/stretchr/testify/require"
)

// PendingBlockAddedTx observers can retain the public coverage result before
// the worker gives it to the corpus. Removing the result entry cannot revoke
// those references, and later transactions must not recycle their maps.
func TestRetainedCoverageAfterCorpusUpdate(t *testing.T) {
	t.Parallel()
	total := coverage.NewCoverageMaps()
	tracer := coverage.NewCoverageTracer()
	var retainedResults []*coverage.CoverageMaps
	for range 100 {
		tracer.OnTxStart(nil, nil, common.Address{})
		result := &chainTypes.MessageResults{AdditionalResults: make(map[string]any)}
		tracer.CaptureTxEndSetAdditionalResults(result)
		retained := coverage.GetCoverageTracerResults(result)
		_, err := retained.UpdateAt(common.Address{1}, common.Hash{1}, 42)
		require.NoError(t, err)
		seq := calls.CallSequence{&calls.CallSequenceElement{ChainReference: &calls.CallSequenceElementChainReference{
			Block: &chainTypes.Block{MessageResults: []*chainTypes.MessageResults{result}},
		}}}
		_, err = checkSequenceCoverageAndUpdate(seq, total)
		require.NoError(t, err)
		require.Nil(t, coverage.GetCoverageTracerResults(result))
		retainedResults = append(retainedResults, retained)
		for _, previous := range retainedResults {
			require.EqualValues(t, 1, previous.BranchesHit(), "retained result was invalidated")
		}
	}
}
