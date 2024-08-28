package reversion

import (
	"fmt"
	"github.com/crytic/medusa/compilation/abiutils"
	fuzzerTypes "github.com/crytic/medusa/fuzzing/contracts"
)

type functionSelector = [4]byte
type errorSelector = [4]byte

var nilSelector = errorSelector{0, 0, 0, 0}

type RevertReport struct {
	RevertedCallReasons map[functionSelector]map[errorSelector]uint
	RevertedCalls       map[functionSelector]uint
	TotalCalls          map[functionSelector]uint
}

func createRevertReport() *RevertReport {
	return &RevertReport{
		RevertedCallReasons: make(map[functionSelector]map[errorSelector]uint),
		RevertedCalls:       make(map[functionSelector]uint),
		TotalCalls:          make(map[functionSelector]uint),
	}
}

func (s *RevertReport) initFuncSelectorIfMissing(selector functionSelector) {
	_, ok := s.TotalCalls[selector]
	if !ok {
		s.TotalCalls[selector] = 0
		s.RevertedCalls[selector] = 0
		s.RevertedCallReasons[selector] = make(map[errorSelector]uint)
	}
}

func (s *RevertReport) initErrSelectorIfMissing(funcSelector functionSelector, errSelector errorSelector) {
	_, ok := s.RevertedCallReasons[funcSelector][errSelector]
	if !ok {
		s.RevertedCallReasons[funcSelector][errSelector] = 0
	}
}

func (s *RevertReport) addCall(funcSelector functionSelector, errSelector errorSelector, didRevert bool) {
	s.addCallCount(funcSelector, errSelector, didRevert, 1)
}

func (s *RevertReport) addCallCount(funcSelector functionSelector, errSelector errorSelector, didRevert bool, number uint) {
	s.initFuncSelectorIfMissing(funcSelector)
	s.TotalCalls[funcSelector] += number
	if didRevert {
		s.RevertedCalls[funcSelector] += number
		s.initErrSelectorIfMissing(funcSelector, errSelector)
		s.RevertedCallReasons[funcSelector][errSelector] += number
	}
}

// Subsumes the data from the `other` report into the receiver report.
func (s *RevertReport) concatReports(other *RevertReport) {
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

// ToArtifact Converts the revert report to an artifact object. Does not populate previous report data or sort the data.
func (s *RevertReport) ToArtifact(contractDefs fuzzerTypes.Contracts) *ReportArtifact {
	funcLookup, errLookup := buildSelectorLookups(contractDefs)
	artifact := &ReportArtifact{}

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

func (s *RevertReport) PrintStats(funcLookup, errLookup map[[4]byte]string) {
	for funcSel, runCount := range s.TotalCalls {
		funcName := funcLookup[funcSel]
		revertCount := s.RevertedCalls[funcSel]
		revertPct := float32(revertCount) / float32(runCount)
		fmt.Printf("%s called %d times. %0.1fpct reverted\n", funcName, runCount, revertPct*100)
		for errSel, errCount := range s.RevertedCallReasons[funcSel] {
			errName, ok := errLookup[errSel]
			if !ok {
				errName = decodeSmuggledSolidityRevertReason(errSel)
			}
			revertPct = float32(errCount) / float32(revertCount)
			fmt.Printf("-> %0.1fpct due to %s\n", revertPct*100, errName)
		}
	}
}

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

func decodeSmuggledSolidityRevertReason(selector [4]byte) string {
	if selector[0] == 0 && selector[1] == 0 && selector[2] == 0 {
		lastByte := selector[3]
		switch lastByte {
		case abiutils.PanicCodeCompilerInserted:
			return "Solidity: genericPanic"
		case abiutils.PanicCodeAssertFailed:
			return "Solidity: assertFailure"
		case abiutils.PanicCodeArithmeticUnderOverflow:
			return "Solidity: uncheckedOver/Underflow"
		case abiutils.PanicCodeDivideByZero:
			return "Solidity: Divide or Mod by zero"
		case abiutils.PanicCodeEnumTypeConversionOutOfBounds:
			return "Solidity: Converted too large a value into enum type"
		case abiutils.PanicCodeIncorrectStorageAccess:
			return "Solidity: Accessed a storage byte array with incorrect encoding"
		case abiutils.PanicCodePopEmptyArray:
			return "Solidity: Called .pop() on empty array"
		case abiutils.PanicCodeOutOfBoundsArrayAccess:
			return "Solidity: Accessed array with out of bounds index"
		case abiutils.PanicCodeAllocateTooMuchMemory:
			return "Solidity: Allocated too much memory"
		case abiutils.PanicCodeCallUninitializedVariable:
			return "Solidity: Called a zero-initialized variable of internal function type"
		}
	}
	return fmt.Sprintf("Unknown %v", selector)
}
