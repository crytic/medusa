package reversion

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	fuzzerTypes "github.com/crytic/medusa/fuzzing/contracts"
	"github.com/crytic/medusa/logging"
	"html/template"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

var (
	//go:embed reversion_template.gohtml
	htmlReportTemplate []byte
)

type ReportArtifact struct {
	FunctionArtifacts []*FunctionArtifact `json:"function_artifacts"`
}

type FunctionArtifact struct {
	Name          string                  `json:"name"`
	TotalCalls    uint                    `json:"total_calls"`
	TotalReverts  uint                    `json:"total_reverts"`
	RevertPct     float64                 `json:"revert_pct"`
	PrevRevertPct *float64                `json:"prev_revert_pct"`
	RevertReasons []*RevertReasonArtifact `json:"revert_reasons"`
}

type RevertReasonArtifact struct {
	Reason            string   `json:"reason"`
	Total             uint     `json:"total"`
	PctAttributed     float64  `json:"pct_attributed"`
	PrevPctAttributed *float64 `json:"prev_pct_attributed"`
}

func CreateRevertReportArtifact(logger *logging.Logger, report *RevertReport, contractDefs fuzzerTypes.Contracts, corpusDir string) (*ReportArtifact, error) {
	prevReportArtifact, err := loadArtifact(logger, corpusDir)
	if err != nil {
		return nil, err
	}

	artifact := convertReportToArtifact(report, contractDefs, prevReportArtifact)
	return artifact, nil
}

func (r *ReportArtifact) ConvertToHtml(buf io.Writer) error {

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
				return fmt.Sprintf("No Change")
			}
		},
	}

	tmpl, err := template.New("revert_stats.html").Funcs(functionMap).Parse(string(htmlReportTemplate))
	if err != nil {
		return fmt.Errorf("could not export report, failed to parse report template: %v", err)
	}

	err = tmpl.Execute(buf, r)
	return err
}

func (r *ReportArtifact) getFunctionArtifact(funcName string) *FunctionArtifact {
	for _, f := range r.FunctionArtifacts {
		if f.Name == funcName {
			return f
		}
	}
	return nil
}

func (r *FunctionArtifact) getRevertReasonArtifact(revertReason string) *RevertReasonArtifact {
	for _, r := range r.RevertReasons {
		if r.Reason == revertReason {
			return r
		}
	}
	return nil
}

func loadArtifact(logger *logging.Logger, dir string) (*ReportArtifact, error) {
	if dir == "" {
		return &ReportArtifact{}, nil
	}

	filePath := filepath.Join(dir, "revert_stats.json")
	b, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("No previous reversion statistics found at ", filePath)
			return &ReportArtifact{}, nil
		} else {
			return nil, err
		}
	}

	var artifact ReportArtifact
	err = json.Unmarshal(b, &artifact)
	if err != nil {
		return nil, err
	}
	return &artifact, nil
}

// updateRevertReasons is used to populate the PrevPctAttributed for function artifacts.
func updateRevertReasons(prevFuncArtifact, newFuncArtifact *FunctionArtifact) {
	for _, prevRevertReason := range prevFuncArtifact.RevertReasons {
		newRevertReason := newFuncArtifact.getRevertReasonArtifact(prevRevertReason.Reason)
		prevPctAttrib := prevRevertReason.PctAttributed
		if newRevertReason == nil {
			if prevRevertReason.Total > 0 {
				// If the new artifact is missing this revert reason, it may mean some code changes made by the user
				// now prevent the revert from occurring. We need to ensure this is flagged
				// in the report, so we'll add a revert reason to the new artifact with 0 hits.
				newRevertReason = &RevertReasonArtifact{
					Reason:            prevRevertReason.Reason,
					Total:             0,
					PctAttributed:     0,
					PrevPctAttributed: &prevPctAttrib,
				}
				newFuncArtifact.RevertReasons = append(newFuncArtifact.RevertReasons, newRevertReason)
			}
		} else {
			newRevertReason.PrevPctAttributed = &prevPctAttrib
		}
	}
}

// convertReportToArtifact takes a report, and converts it to an artifact with fully populated previous run metrics
func convertReportToArtifact(
	stats *RevertReport,
	contractDefs fuzzerTypes.Contracts,
	prevArtifact *ReportArtifact) *ReportArtifact {

	artifact := stats.ToArtifact(contractDefs)

	// iterate over the fields of the previous artifact to populate the prev fields of the new artifact
	for _, prevFuncArtifact := range prevArtifact.FunctionArtifacts {
		newFuncArtifact := artifact.getFunctionArtifact(prevFuncArtifact.Name)
		if newFuncArtifact == nil {
			continue
		}
		// prevent looppointer issue
		prevFuncRevertPct := prevFuncArtifact.RevertPct
		newFuncArtifact.PrevRevertPct = &prevFuncRevertPct
		updateRevertReasons(prevFuncArtifact, newFuncArtifact)

		// sort the revert reasons while we're here
		sort.Slice(newFuncArtifact.RevertReasons, func(i, j int) bool {
			return newFuncArtifact.RevertReasons[i].Reason < newFuncArtifact.RevertReasons[j].Reason
		})
	}
	// sort the functions by name
	sort.Slice(artifact.FunctionArtifacts, func(i, j int) bool {
		return artifact.FunctionArtifacts[i].Name < artifact.FunctionArtifacts[j].Name
	})
	return artifact
}
