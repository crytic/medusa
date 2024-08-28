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

type ReversionMeasurer struct {
	incomingReportsQueue chan *RevertReport

	aggReport      *RevertReport
	enabled        bool
	writeReports   bool
	reportArtifact *ReportArtifact
}

// CreateReversionMeasurer creates a new ReversionMeasurer using the provided config
func CreateReversionMeasurer(config config.ProjectConfig) *ReversionMeasurer {
	return &ReversionMeasurer{
		incomingReportsQueue: make(chan *RevertReport, 100),
		aggReport:            createRevertReport(),
		enabled:              config.Fuzzing.Testing.ReversionMeasurement.Enabled,
		writeReports:         config.Fuzzing.Testing.ReversionMeasurement.WriteReports,
	}
}

// StartWorker creates a background goroutine that handles incoming reversion reports from workers.
// The background goroutine is terminated when the provided context is terminated, or when the
// cleanup function returned by StartWorker is called.
func (s *ReversionMeasurer) StartWorker(ctx context.Context) func() {
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

// OnPendingBlockCommittedEvent is used to identify top level calls made by the fuzzer, extract the result of those
// calls, then add the data to a reversion report that will be submitted to the background worker via incomingReportsQueue
func (s *ReversionMeasurer) OnPendingBlockCommittedEvent(event chain.PendingBlockCommittedEvent) error {
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

// BuildArtifact converts aggregated report information into an artifact that can be easily serialized.
func (s *ReversionMeasurer) BuildArtifact(logger *logging.Logger, contractDefs fuzzerTypes.Contracts, corpusDir string) error {
	if !s.enabled || !s.writeReports {
		return nil
	}
	artifact, err := CreateRevertReportArtifact(logger, s.aggReport, contractDefs, corpusDir)
	if err != nil {
		return err
	}

	s.reportArtifact = artifact
	return nil
}

// WriteReport takes the generated reportArtifact and writes it to the provided dir. Two artifacts are written;
// a revert statistics json file, and a user-readable html file.
func (s *ReversionMeasurer) WriteReport(dir string, logger *logging.Logger) error {
	if !s.enabled || !s.writeReports {
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

// writeReportHtml generates an HTML representation of the report artifact and writes it to path.
func (s *ReversionMeasurer) writeReportHtml(path string, logger *logging.Logger) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	err = s.reportArtifact.ConvertToHtml(file)

	if err == nil {
		logger.Info("Revert stats report written to: ", path)
	}
	return err
}

// writeReportHtml generates an JSON representation of the report artifact and writes it to path.
func (s *ReversionMeasurer) writeReportJson(path string, logger *logging.Logger) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
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

// getRevertReason encodes the error from `result` into an errorSelector. If the error is a solidity-inserted revert,
// the returnData is decoded to identify the type of revert, and this id is smuggled into the last byte of errorSelector.
// These smuggled errors can be decoded using decodeSmuggledSolidityRevertReason.
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
