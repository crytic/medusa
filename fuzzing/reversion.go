package fuzzing

import (
	"context"
	"errors"
	"fmt"
	fuzzerTypes "github.com/crytic/medusa/fuzzing/contracts"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
)

type ReversionStatistics struct {
	//contractDefinitions fuzzerTypes.Contracts
	incomingReportsQueue chan *SequenceRevertReport

	aggReport *SequenceRevertReport
}

func CreateReversionStatistics() *ReversionStatistics {
	return &ReversionStatistics{
		incomingReportsQueue: make(chan *SequenceRevertReport, 100),
		aggReport:            createSequencerRevertReport(),
	}
}

type functionSelector = [4]byte
type errorSelector = [4]byte

var nilSelector = errorSelector{0, 0, 0, 0}

type SequenceRevertReport struct {
	RevertedCallReasons map[functionSelector]map[errorSelector]uint
	RevertedCalls       map[functionSelector]uint
	TotalCalls          map[functionSelector]uint
}

func (s *SequenceRevertReport) initFuncSelectorIfMissing(selector functionSelector) {
	_, ok := s.TotalCalls[selector]
	if !ok {
		s.TotalCalls[selector] = 0
		s.RevertedCalls[selector] = 0
		s.RevertedCallReasons[selector] = make(map[errorSelector]uint)
	}
}

func (s *SequenceRevertReport) initErrSelectorIfMissing(funcSelector functionSelector, errSelector errorSelector) {
	_, ok := s.RevertedCallReasons[funcSelector][errSelector]
	if !ok {
		s.RevertedCallReasons[funcSelector][errSelector] = 0
	}
}

func (s *SequenceRevertReport) addCall(funcSelector functionSelector, errSelector errorSelector, didRevert bool) {
	s.addCallCount(funcSelector, errSelector, didRevert, 1)
}

func (s *SequenceRevertReport) addCallCount(funcSelector functionSelector, errSelector errorSelector, didRevert bool, number uint) {
	s.initFuncSelectorIfMissing(funcSelector)
	s.TotalCalls[funcSelector] += number
	if didRevert {
		s.RevertedCalls[funcSelector] += number
		s.initErrSelectorIfMissing(funcSelector, errSelector)
		s.RevertedCallReasons[funcSelector][errSelector] += number
	}
}

// Subsumes the data from the `other` report into the receiver report.
func (s *SequenceRevertReport) concatReports(other *SequenceRevertReport) {
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

func createSequencerRevertReport() *SequenceRevertReport {
	return &SequenceRevertReport{
		RevertedCallReasons: make(map[functionSelector]map[errorSelector]uint),
		RevertedCalls:       make(map[functionSelector]uint),
		TotalCalls:          make(map[functionSelector]uint),
	}
}

func (s *ReversionStatistics) PrintStats(contractDefs fuzzerTypes.Contracts) {
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
	errorLookup[errorSelector{78, 72, 123, 113}] = "Panic()"
	s.aggReport.PrintStats(functionLookup, errorLookup)
}

func (s *SequenceRevertReport) PrintStats(funcLookup, errLookup map[[4]byte]string) {
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

func (s *ReversionStatistics) StartWorker(ctx context.Context) func() {
	workerCtx, done := context.WithCancel(ctx)
	go func() {
		for {
			select {
			case report := <-s.incomingReportsQueue:
				s.aggReport.concatReports(report)
			case <-workerCtx.Done():
				break
			}
		}
	}()
	return done
}

func (s *ReversionStatistics) onFuzzWorkerCallSequenceTestedEvent(event FuzzerWorkerCallSequenceTestedEvent) error {
	report := createSequencerRevertReport()

	for _, call := range event.Sequence {
		if call.ExecutionResult != nil {

			// Disregard out of gas errors, arithmetic underflow, etc.
			// todo: should we respect the revert error types in the medusa configs?
			if call.ExecutionResult.Err != nil && !errors.Is(call.ExecutionResult.Err, vm.ErrExecutionReverted) {
				continue
			}

			if len(call.Call.Data) < 4 {
				continue
			}

			funcSelector := functionSelector{}
			copy(funcSelector[:], call.Call.Data[:4])

			if call.ExecutionResult.Err != nil {
				revertReason := getRevertReason(call.ExecutionResult)
				report.addCall(funcSelector, revertReason, true)
			} else {
				report.addCall(funcSelector, errorSelector{}, false)
			}

		}
	}

	s.incomingReportsQueue <- report
	return nil
}

func getRevertReason(result *core.ExecutionResult) errorSelector {
	revertReason := nilSelector
	if len(result.ReturnData) >= 4 {
		copy(revertReason[:], result.ReturnData[:4])

		// If compiler-inserted revert, then smuggle the revert reason out
		if revertReason[0] == 78 && revertReason[1] == 72 && revertReason[2] == 123 && revertReason[3] == 113 {
			revertReason[0] = 0
			revertReason[1] = 0
			revertReason[2] = 0
			revertReason[3] = result.ReturnData[35]
		}
	}

	return revertReason
}

func decodeSmuggledSolidityRevertReason(selector [4]byte) string {
	if selector[0] == 0 && selector[1] == 0 && selector[2] == 0 {
		lastByte := selector[3]
		switch lastByte {
		case 0x00:
			return "Solidity: genericPanic"
		case 0x01:
			return "Solidity: assertFailure"
		case 0x11:
			return "Solidity: uncheckedOver/Underflow"
		case 0x12:
			return "Solidity: Divide or Mod by zero"
		case 0x21:
			return "Solidity: Converted too large a value into enum type"
		case 0x22:
			return "Solidity: Accessed a storage byte array with incorrect encoding"
		case 0x31:
			return "Solidity: Called .pop() on empty array"
		case 0x32:
			return "Solidity: Accessed array with out of bounds index"
		case 0x41:
			return "Solidity: Allocated too much memory"
		case 0x51:
			return "Solidity: Called a zero-initialized variable of internal function type"
		}
	}
	return fmt.Sprintf("Unknown %v", selector)
}
