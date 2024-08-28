package reversion

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/compilation/abiutils"
	"github.com/crytic/medusa/fuzzing/config"
	fuzzerTypes "github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/logging"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"os"
	"path/filepath"
)

type ReversionStatistics struct {
	incomingReportsQueue chan *RevertReport

	aggReport       *RevertReport
	enabled         bool
	displayAfterRun bool
	reportArtifact  *ReportArtifact
}

func CreateReversionStatistics(config config.ProjectConfig) *ReversionStatistics {
	return &ReversionStatistics{
		incomingReportsQueue: make(chan *RevertReport, 100),
		aggReport:            createRevertReport(),
		enabled:              config.Fuzzing.Testing.ReversionMeasurement.Enabled,
		displayAfterRun:      config.Fuzzing.Testing.ReversionMeasurement.DisplayAfterRun,
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
	if !s.enabled {
		return nil
	}
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

func (s *ReversionStatistics) BuildArtifactAndPrintResults(logger *logging.Logger, contractDefs fuzzerTypes.Contracts, corpusDir string) error {
	if !s.enabled {
		return nil
	}
	artifact, err := CreateRevertReportArtifact(logger, s.aggReport, contractDefs, corpusDir)
	if err != nil {
		return err
	}

	s.reportArtifact = artifact
	if s.displayAfterRun {
		// display
	}

	return nil
}

func (s *ReversionStatistics) WriteReport(dir string, logger *logging.Logger) error {
	if !s.enabled {
		return nil
	}
	if s.reportArtifact == nil {
		return errors.New("report artifact missing")
	}
	jsonpath := filepath.Join(dir, "revert_stats.json")
	err := s.writeReportJson(jsonpath, logger)
	if err != nil {
		return err
	}

	markdownpath := filepath.Join(dir, "revert_stats.html")
	err = s.writeReportHtml(markdownpath, logger)

	return err
}

func (s *ReversionStatistics) PrintStats(contractDefs fuzzerTypes.Contracts) {
	if !s.enabled || !s.displayAfterRun {
		return
	}
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
	//errorLookup[errorSelector{78, 72, 123, 113}] = "Panic()"
	s.aggReport.PrintStats(functionLookup, errorLookup)
}

func (s *ReversionStatistics) writeReportHtml(path string, logger *logging.Logger) error {
	file, err := os.Create(path)
	defer file.Close()
	if err != nil {
		return err
	}
	err = s.reportArtifact.ConvertToHtml(file)

	if err == nil {
		logger.Info("Revert stats report written to: ", path)
	}
	return err
}

func (s *ReversionStatistics) writeReportJson(path string, logger *logging.Logger) error {
	file, err := os.Create(path)
	defer file.Close()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.reportArtifact, "", "    ")
	if err != nil {
		return err
	}
	_, err = file.Write(b)
	if err == nil {
		logger.Info("Revert stats written to file: ", path)
	}
	return err
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
