package reversion

import (
	"fmt"
	"github.com/crytic/medusa/compilation/abiutils"
	fuzzerTypes "github.com/crytic/medusa/fuzzing/contracts"
)

type functionSelector = [4]byte
type errorSelector = [4]byte

var nilSelector = errorSelector{0, 0, 0, 0}

// TxCallMetrics is used to track the reversion statistics for a given sequence or fuzzing session.
type TxCallMetrics struct {
	RevertedCallReasons map[functionSelector]map[errorSelector]uint
	RevertedCalls       map[functionSelector]uint
	TotalCalls          map[functionSelector]uint
}

// createTxCallMetrics initializes an empty TxCallMetrics
func createTxCallMetrics() *TxCallMetrics {
	return &TxCallMetrics{
		RevertedCallReasons: make(map[functionSelector]map[errorSelector]uint),
		RevertedCalls:       make(map[functionSelector]uint),
		TotalCalls:          make(map[functionSelector]uint),
	}
}

// initFuncSelectorIfMissing is a utility function used to initialize TxCallMetrics for a specific function selector
func (s *TxCallMetrics) initFuncSelectorIfMissing(selector functionSelector) {
	_, ok := s.TotalCalls[selector]
	if !ok {
		s.TotalCalls[selector] = 0
		s.RevertedCalls[selector] = 0
		s.RevertedCallReasons[selector] = make(map[errorSelector]uint)
	}
}

// initErrSelectorIfMissing is a utility function used to initialize RevertedCallReasons for a specific error selector
func (s *TxCallMetrics) initErrSelectorIfMissing(funcSelector functionSelector, errSelector errorSelector) {
	_, ok := s.RevertedCallReasons[funcSelector][errSelector]
	if !ok {
		s.RevertedCallReasons[funcSelector][errSelector] = 0
	}
}

// addCall is used to add a single call to the revert report
func (s *TxCallMetrics) addCall(funcSelector functionSelector, errSelector errorSelector, didRevert bool) {
	s.addCallCount(funcSelector, errSelector, didRevert, 1)
}

// addCallCount is used to add one or more calls to the revert report.
func (s *TxCallMetrics) addCallCount(funcSelector functionSelector, errSelector errorSelector, didRevert bool, number uint) {
	s.initFuncSelectorIfMissing(funcSelector)
	s.TotalCalls[funcSelector] += number
	if didRevert {
		s.RevertedCalls[funcSelector] += number
		s.initErrSelectorIfMissing(funcSelector, errSelector)
		s.RevertedCallReasons[funcSelector][errSelector] += number
	}
}

// concatReports Subsumes the data from the `other` report into the receiver report.
func (s *TxCallMetrics) concatReports(other *TxCallMetrics) {
	for fSel, callCount := range other.TotalCalls {
		revertCount := other.RevertedCalls[fSel]
		revertReasons := other.RevertedCallReasons[fSel]

		for revReason, revCount := range revertReasons {
			s.addCallCount(fSel, revReason, true, revCount)
		}

		successCount := callCount - revertCount
		s.addCallCount(fSel, errorSelector{}, false, successCount)
	}
}

// ToRevertArtifact Converts the revert report to a revert artifact object. Does not populate previous report data or sort the data.
func (s *TxCallMetrics) ToRevertArtifact(contractDefs fuzzerTypes.Contracts) *RevertArtifact {
	funcLookup, errLookup := buildSelectorLookups(contractDefs)
	artifact := &RevertArtifact{}

	for funcSel, runCount := range s.TotalCalls {
		revertCount := s.RevertedCalls[funcSel]
		funcName := funcLookup[funcSel]
		funcArtifact := &FunctionArtifact{
			Name:          funcName,
			TotalCalls:    runCount,
			TotalReverts:  revertCount,
			RevertPct:     float64(revertCount) / float64(runCount),
			PrevRevertPct: nil,
			RevertReasons: []*RevertReasonArtifact{},
		}

		for errSel, errCount := range s.RevertedCallReasons[funcSel] {
			errName, ok := errLookup[errSel]
			if !ok {
				errName = decodeSmuggledSolidityRevertReason(errSel)
			}
			revertReason := &RevertReasonArtifact{
				Reason:            errName,
				Total:             errCount,
				PctAttributed:     float64(errCount) / float64(revertCount),
				PrevPctAttributed: nil,
			}
			funcArtifact.RevertReasons = append(funcArtifact.RevertReasons, revertReason)
		}
		artifact.FunctionArtifacts = append(artifact.FunctionArtifacts, funcArtifact)
	}
	return artifact
}

// buildSelectorLookups is used to construct a selector->string lookup based on the provided contractDefs
func buildSelectorLookups(contractDefs fuzzerTypes.Contracts) (map[functionSelector]string, map[errorSelector]string) {
	errorLookup := make(map[errorSelector]string)
	functionLookup := make(map[functionSelector]string)

	for _, contract := range contractDefs {
		for label, detail := range contract.CompiledContract().Abi.Errors {
			errSel := errorSelector{}
			copy(errSel[:], detail.ID[:4])
			errorLookup[errSel] = label
		}
		for label, detail := range contract.CompiledContract().Abi.Methods {
			funcSel := functionSelector{}
			copy(funcSel[:], detail.ID[:4])
			functionLookup[funcSel] = label
		}
	}
	return functionLookup, errorLookup
}

// decodeSmuggledSolidityRevertReason is used to decode solidity-inserted reverts, and as a backstop for error selectors
// that we do not have an ABI for.
func decodeSmuggledSolidityRevertReason(selector [4]byte) string {
	if selector[0] == 0 && selector[1] == 0 && selector[2] == 0 {
		lastByte := selector[3]
		return abiutils.GetPanicReason(uint64(lastByte))
	}
	return fmt.Sprintf("Unknown %v", selector)
}
