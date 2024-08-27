package reversion

import (
	"context"
	"errors"
	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/compilation/abiutils"
	fuzzerTypes "github.com/crytic/medusa/fuzzing/contracts"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
)

type ReversionStatistics struct {
	incomingReportsQueue chan *RevertReport

	aggReport *RevertReport
}

func CreateReversionStatistics() *ReversionStatistics {
	return &ReversionStatistics{
		incomingReportsQueue: make(chan *RevertReport, 100),
		aggReport:            createRevertReport(),
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

func (s *ReversionStatistics) OnPendingBlockCommittedEvent(event chain.PendingBlockCommittedEvent) error {
	report := createRevertReport()

	for i, msg := range event.Block.Messages {
		msgResult := event.Block.MessageResults[i]

		// Disregard out of gas errors, etc.
		if msgResult.ExecutionResult.Err != nil && !errors.Is(msgResult.ExecutionResult.Err, vm.ErrExecutionReverted) {
			continue
		}

		// Disregard deployments
		if msg.To == nil {
			continue
		}

		if len(msg.Data) < 4 {
			continue
		}

		funcSelector := functionSelector{}
		copy(funcSelector[:], msg.Data[:4])

		if msgResult.ExecutionResult.Err != nil {
			revertReason := getRevertReason(msgResult.ExecutionResult)
			report.addCall(funcSelector, revertReason, true)
		} else {
			report.addCall(funcSelector, errorSelector{}, false)
		}
	}

	s.incomingReportsQueue <- report
	return nil
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

func getRevertReason(result *core.ExecutionResult) errorSelector {
	revertReason := nilSelector
	if len(result.ReturnData) >= 4 {
		copy(revertReason[:], result.ReturnData[:4])

		// If compiler-inserted revert, then smuggle the revert reason out
		panicCode := abiutils.GetSolidityPanicCode(result.Err, result.ReturnData, true)
		if panicCode != nil {
			revertReason[0] = 0
			revertReason[1] = 0
			revertReason[2] = 0
			revertReason[3] = panicCode.Bytes()[0]
		}
	}
	return revertReason
}
