package reverts

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/logging"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	//go:embed report_template.gohtml
	htmlReportTemplate []byte
)

// RevertReporter is responsible for tracking and storing the RevertMetrics for the fuzzing campaign.
type RevertReporter struct {
	// Enabled determines if the reporter is enabled
	Enabled bool

	// Path is the path to the directory where the revert metrics and report will be stored from this campaign.
	// We will also look at this directory for the RevertMetricsArtifact from the previous campaign.
	Path string

	// CustomErrors is a map of different error IDs to their custom errors. This is used to resolve the revert reasons
	// when updating the revert metrics.
	CustomErrors map[string]abi.Error

	// RevertMetricsCh is the channel that will receive RevertMetricsUpdate objects from fuzzer workers to update the revert metrics.
	// We will be receiving pointers to the execution result object within the RevertMetricsUpdate object since we are confident that
	// will be no races on that value.
	RevertMetricsCh chan RevertMetricsUpdate

	// RevertMetrics holds the revert metrics for the current campaign.
	RevertMetrics *RevertMetrics

	// PrevRevertMetrics holds the revert metrics for the previous campaign.
	PrevRevertMetrics *RevertMetrics
}

// NewRevertReporter creates a new RevertsReporter. If there is any issue loading the previous artifact (if it exists),
// an error is returned.
func NewRevertReporter(enabled bool, corpusDirectory string) (*RevertReporter, error) {
	if !enabled {
		return &RevertReporter{}, nil
	}
	// Generate the path based on whether the corpus directory is empty or not
	var path string
	if corpusDirectory == "" {
		path = filepath.Join("crytic-export", "coverage")
	} else {
		path = filepath.Join(corpusDirectory, "coverage")
	}

	// Try to load the revert metrics from the previous campaign
	prevRevertMetrics, err := NewRevertMetricsFromPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load previous revert metrics: %v", err)
	}

	return &RevertReporter{
		Enabled:           enabled,
		Path:              path,
		RevertMetrics:     NewRevertMetrics(),
		PrevRevertMetrics: prevRevertMetrics,
		// We are going to make a buffered channel here to avoid blocking the worker.
		// Praying that 1000 is enough to avoid any issues.
		RevertMetricsCh: make(chan RevertMetricsUpdate, 1000),
	}, nil
}

// AddCustomErrors allows users to provide a list of contract definitions that will be parsed to identify any custom,
// user-defined errors in each contract's ABI. We _need_ to decouple this from the creation of the actual RevertReporter
// object because of some edge case behavior with using crytic-export as the output folder for revert reports.
// Since the crytic-export folder is deleted by crytic-compile, we need to create the revert reporter, compile,
// and then attach the compiled artifacts to perform comparative analysis to the previous fuzzing campaign.
func (r *RevertReporter) AddCustomErrors(contractDefinitions contracts.Contracts) {
	// Guard clause
	if !r.Enabled {
		return
	}

	// Create the custom errors mapping
	r.CustomErrors = make(map[string]abi.Error)

	// Iterate over the contract definitions and get the error IDs
	for _, contract := range contractDefinitions {
		// Iterate over the errors in the contract's ABI
		for _, err := range contract.CompiledContract().Abi.Errors {
			// Add the error ID to the map (first four bytes)
			errID := strings.TrimPrefix(err.ID.Hex(), "0x")
			r.CustomErrors[errID[:8]] = err
		}
	}
}

// Start starts the revert reporter goroutine. It will continue to run until the context is cancelled.
func (r *RevertReporter) Start(ctx context.Context) {
	// Don't do anything if the revert reporter is not enabled
	if !r.Enabled {
		return
	}

	// Start the revert metrics update loop
	go func() {
		for {
			select {
			case <-ctx.Done():
				// If the context is done, we need to exit the goroutine
				return
			case update := <-r.RevertMetricsCh:
				// Update the revert metrics
				r.RevertMetrics.Update(&update, r.CustomErrors)
			}
		}
	}()
}

// BuildArtifacts writes the revert metrics to disk and also generates an HTML version of it for the user.
func (r *RevertReporter) BuildArtifacts() error {
	if !r.Enabled {
		return nil
	}

	// If we have a previous revert metrics, we need to merge them with the current revert metrics
	r.RevertMetrics.Finalize(r.PrevRevertMetrics)

	// Write JSON report to disk
	err := r.writeJSONReport()
	if err != nil {
		return err
	}

	// Write HTML report to disk
	err = r.writeHTMLReport()
	return err
}

// writeJSONReport generates a JSON representation of the report metrics and writes it to disk.
func (r *RevertReporter) writeJSONReport() error {
	// Calculate path
	path := filepath.Join(r.Path, "revert_report.json")

	// Update the indentation
	b, err := json.MarshalIndent(r.RevertMetrics, "", "\t")
	if err != nil {
		return err
	}

	// Write to file
	err = os.WriteFile(path, b, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON revert report at %v: %v", path, err)
	}

	logging.GlobalLogger.Info("JSON revert report saved to: ", path)
	return nil
}

// writeHTMLReport generates an HTML representation of the revert metrics and writes it to disk.
func (r *RevertReporter) writeHTMLReport() error {
	// Calculate path and create file
	path := filepath.Join(r.Path, "revert_report.html")
	file, err := os.Create(path)
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to open HTML revert report for writing: %v", err)
	}
	defer file.Close()

	functionMap := template.FuncMap{
		"timeNow": time.Now,
		"statSigThresh": func() int {
			return 100
		},
		"add": func(x int, y int) int {
			return x + y
		},
		"relativePath": func(path string) string {
			// Obtain a path relative to our current working directory.
			// If we encounter an error, return the original path.
			cwd, err := os.Getwd()
			if err != nil {
				return path
			}
			relativePath, err := filepath.Rel(cwd, path)
			if err != nil {
				return path
			}

			return relativePath
		},
		"percentageStr": func(x int, y int, decimals int) string {
			// Determine our precision string
			formatStr := "%." + strconv.Itoa(decimals) + "f"

			// If no lines are active and none are covered, show 0% coverage
			if x == 0 && y == 0 {
				return fmt.Sprintf(formatStr, float64(0))
			}
			return fmt.Sprintf(formatStr, (float64(x)/float64(y))*100)
		},
		"percentageInt": func(x int, y int) int {
			if y == 0 {
				return 100
			}
			return int(math.Round(float64(x) / float64(y) * 100))
		},
		"percentageFmt": func(x float64) string {
			return fmt.Sprintf("%0.1f%%", x*100)
		},
		"percentageFmtOpt": func(x *float64) string {
			if x == nil {
				return "No prev. data"
			} else {
				return fmt.Sprintf("%0.1f%%", *x*100)
			}
		},
		"percentageChangeOpt": func(v0 *float64, v1 float64) string {
			if v0 == nil {
				return "No prev. data"
			}
			val := (v1 - *v0) / math.Abs(*v0) * 100
			if val > 0 {
				return fmt.Sprintf("Increased by %.1f%%", val)
			}
			if val < 0 {
				return fmt.Sprintf("Decreased by %.1f%%", -val)
			} else {
				return "No Change"
			}
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
		"floatPtr": func(f float64) *float64 {
			return &f
		},
	}

	tmpl, err := template.New("revert_report.html").Funcs(functionMap).Parse(string(htmlReportTemplate))
	if err != nil {
		return fmt.Errorf("failed to parse HTML revert report template: %v", err)
	}

	err = tmpl.Execute(file, r.RevertMetrics)
	if err != nil {
		return fmt.Errorf("failed to write HTML revert report at %v: %v", path, err)
	}

	logging.GlobalLogger.Info("HTML revert report written to: ", path)
	return nil
}
