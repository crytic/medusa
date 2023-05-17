package coverage

import (
	_ "embed"
	"fmt"
	"github.com/crytic/medusa/compilation/types"
	"github.com/crytic/medusa/utils"
	"html/template"
	"os"
	"path/filepath"
	"time"
)

var (
	//go:embed report_template.gohtml
	htmlReportTemplate []byte
)

// GenerateReport takes a set of CoverageMaps and compilations, and produces a coverage report using them, detailing
// all source mapped ranges of the source files which were covered or not.
// Returns an error if one occurred.
func GenerateReport(compilations []types.Compilation, coverageMaps *CoverageMaps, htmlReportPath string) error {
	// Perform source analysis.
	sourceAnalysis, err := AnalyzeSourceCoverage(compilations, coverageMaps)
	if err != nil {
		return err
	}

	// Finally, export the report data we analyzed.
	if htmlReportPath != "" {
		err = exportCoverageReport(sourceAnalysis, htmlReportPath)
	}
	return err
}

// exportCoverageReport takes a previously performed source analysis and generates an HTML coverage report from it.
// Returns an error if one occurs.
func exportCoverageReport(sourceAnalysis *SourceAnalysis, outputPath string) error {
	// Copy all source files from our analysis into a list.
	sourceFiles := make([]*SourceFileAnalysis, 0)
	for filePath, sourceFile := range sourceAnalysis.Files {
		sourceFiles = append(sourceFiles, &SourceFileAnalysis{
			Path:  filePath,
			Lines: sourceFile.Lines,
		})
	}

	// Define mappings onto some useful variables/functions.
	functionMap := template.FuncMap{
		"timeNow": time.Now,
		"add": func(x int, y int) int {
			return x + y
		},
		"percentageStr": func(x int, y int) string {
			return fmt.Sprintf("%.1f", (float64(x)/float64(y))*100)
		},
		"sourceLinesCovered": func(sourceFile *SourceFileAnalysis) int {
			coveredCount := 0
			for _, sourceLine := range sourceFile.Lines {
				if sourceLine.IsCovered || sourceLine.IsCoveredReverted {
					coveredCount++
				}
			}
			return coveredCount
		},
		"sourceLinesActive": func(sourceFile *SourceFileAnalysis) int {
			activeCount := 0
			for _, sourceLine := range sourceFile.Lines {
				if sourceLine.IsActive {
					activeCount++
				}
			}
			return activeCount
		},
	}

	// Parse our HTML template
	tmpl, err := template.New("coverage_report.html").Funcs(functionMap).Parse(string(htmlReportTemplate))
	if err != nil {
		return fmt.Errorf("could not export report, failed to parse report template: %v", err)
	}

	// If the parent directory doesn't exist, create it.
	parentDirectory := filepath.Dir(outputPath)
	err = utils.MakeDirectory(parentDirectory)
	if err != nil {
		return err
	}

	// Create our report file
	file, err := os.Create(outputPath)
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("could not export report, failed to open file for writing: %v", err)
	}

	// Execute the template and write it back to file.
	err = tmpl.Execute(file, sourceFiles)
	fileCloseErr := file.Close()
	if err == nil {
		err = fileCloseErr
	}
	return err
}
