package coverage

import (
	_ "embed"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/crytic/medusa/compilation/types"
	"github.com/crytic/medusa/utils"
)

var (
	//go:embed report_template.gohtml
	htmlReportTemplate []byte
)

// GenerateReports takes a set of CoverageMaps and compilations, and produces a coverage report using them, detailing
// all source mapped ranges of the source files which were covered or not.
// Returns an error if one occurred.
func GenerateReports(compilations []types.Compilation, coverageMaps *CoverageMaps, reportDir string) error {
	// Perform source analysis.
	sourceAnalysis, err := AnalyzeSourceCoverage(compilations, coverageMaps)
	if err != nil {
		return err
	}

	if reportDir != "" {
		// Save the LCOV report.
		err = saveLCOVReport(sourceAnalysis, reportDir)
		if err != nil {
			return err
		}

		// Save the HTML report.
		err = saveHTMLReport(sourceAnalysis, reportDir)
	}
	return err
}

// saveHTMLReport takes a previously performed source analysis and generates an HTML coverage report from it.
func saveHTMLReport(sourceAnalysis *SourceAnalysis, reportDir string) error {
	// Define mappings onto some useful variables/functions.
	functionMap := template.FuncMap{
		"timeNow": time.Now,
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
	}

	// Parse our HTML template
	tmpl, err := template.New("coverage_report.html").Funcs(functionMap).Parse(string(htmlReportTemplate))
	if err != nil {
		return fmt.Errorf("could not export report, failed to parse report template: %v", err)
	}

	// If the directory doesn't exist, create it.
	err = utils.MakeDirectory(reportDir)
	if err != nil {
		return err
	}

	// Create our report file
	htmlReportPath := filepath.Join(reportDir, "coverage_report.html")
	file, err := os.Create(htmlReportPath)
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("could not export report, failed to open file for writing: %v", err)
	}

	// Execute the template and write it back to file.
	err = tmpl.Execute(file, sourceAnalysis)
	fileCloseErr := file.Close()
	if err == nil {
		err = fileCloseErr
	}
	return err
}

// saveLCOVReport takes a previously performed source analysis and generates an LCOV report from it.
func saveLCOVReport(sourceAnalysis *SourceAnalysis, reportDir string) error {
	// Generate the LCOV report.
	lcovReport := sourceAnalysis.GenerateLCOVReport()

	// If the directory doesn't exist, create it.
	err := utils.MakeDirectory(reportDir)
	if err != nil {
		return err
	}

	// Write the LCOV report to a file.
	lcovReportPath := filepath.Join(reportDir, "lcov.info")
	err = os.WriteFile(lcovReportPath, []byte(lcovReport), 0644)
	if err != nil {
		return fmt.Errorf("could not export LCOV report: %v", err)
	}

	return nil
}
