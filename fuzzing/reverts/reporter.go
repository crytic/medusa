package reverts

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/crytic/medusa/chain"
	"github.com/crytic/medusa/compilation/abiutils"
	fuzzerTypes "github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/logging"
	"github.com/crytic/medusa/utils"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"os"
	"path/filepath"
)

// RevertReporter is responsible for tracking and reporting revert stats across the fuzzer workers. These revert
// statistics are then provided as an artifact to the end user.
type RevertReporter struct {
	incomingMetricsQueue chan *TxCallMetrics
	aggregatedMetrics    *TxCallMetrics
	reportArtifact       *RevertArtifact
	cleanupWorker        func()
}

// NewRevertsReporter creates a new RevertsReporter
func NewRevertsReporter() *RevertReporter {
	return &RevertReporter{
		incomingMetricsQueue: make(chan *TxCallMetrics, 500),
		aggregatedMetrics:    createTxCallMetrics(),
	}
}

// Start creates a background goroutine that handles incoming reversion reports from workers.
// The background goroutine is terminated when the provided context is terminated, or when the
// cleanup function returned by StartWorker is called.
func (s *RevertReporter) Start(ctx context.Context) {
	for !utils.CheckContextDone(ctx) {
		select {
		case report := <-s.incomingMetricsQueue:
			s.aggregatedMetrics.concatReports(report)
		default:
		}
	}
	// TODO: Maybe do some cleanup?
}

// OnPendingBlockCommittedEvent is used to identify top level calls made by the fuzzer, extract the result of those
// calls, then add the data to a reversion report that will be submitted to the background worker via incomingMetricsQueue
func (s *RevertReporter) OnPendingBlockCommittedEvent(event chain.PendingBlockCommittedEvent) error {
	report := createTxCallMetrics()

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

	s.incomingMetricsQueue <- report
	return nil
}

// BuildArtifact converts aggregated report information into an artifact that can be easily serialized.
func (s *RevertReporter) BuildArtifact(logger *logging.Logger, contractDefs fuzzerTypes.Contracts, corpusDir string) error {
	// terminate the worker to make sure all the stats are aggregated
	s.cleanupWorker()

	artifact, err := CreateRevertArtifact(logger, s.aggregatedMetrics, contractDefs, corpusDir)
	if err != nil {
		return err
	}

	s.reportArtifact = artifact
	return nil
}

// WriteReport takes the generated reportArtifact and writes it to the provided dir. Two artifacts are written;
// a revert statistics json file, and a user-readable html file.
func (s *RevertReporter) WriteReport(dir string, logger *logging.Logger) error {
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
func (s *RevertReporter) writeReportHtml(path string, logger *logging.Logger) error {
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
func (s *RevertReporter) writeReportJson(path string, logger *logging.Logger) error {
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
