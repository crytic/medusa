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

	"github.com/crytic/medusa/utils"
)

var (
	//go:embed report_template.gohtml
	htmlReportTemplate []byte
)

// WriteHTMLReport takes a previously performed source analysis and generates an HTML coverage report from it.
func WriteHTMLReport(sourceAnalysis *SourceAnalysis, reportDir string) (string, error) {
	// Define mappings onto some useful variables/functions.
	functionMap := template.FuncMap{
		"formatNumber": func(num uint64) string {
			// Format large numbers to be more readable (e.g., 1234 → 1.2K, 1500000 → 1.5M)
			if num < 1000 {
				return fmt.Sprintf("%d", num) // Keep small numbers as is
			} else if num < 1000000 {
				// Format as K (thousands)
				value := float64(num) / 1000.0
				return fmt.Sprintf("%.1fK", value)
			} else if num < 1000000000 {
				// Format as M (millions)
				value := float64(num) / 1000000.0
				return fmt.Sprintf("%.1fM", value)
			} else {
				// Format as B (billions)
				value := float64(num) / 1000000000.0
				return fmt.Sprintf("%.1fB", value)
			}
		},
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
		"filePathToId": func(path string) string {
			// Convert a file path to a safe HTML ID by replacing non-alphanumeric characters with underscores
			safeId := ""
			for _, c := range path {
				if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
					safeId += string(c)
				} else {
					safeId += "_"
				}
			}
			return safeId
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
		"getCoverageColor": func(percentage int) string {
			// Color gradient: red (0%) -> yellow (50%) -> green (100%)
			if percentage < 50 {
				// Red to yellow (0-50%)
				return fmt.Sprintf("hsl(%d, 90%%, 50%%)", int(float64(percentage)*1.2))
			} else {
				// Yellow to green (50-100%)
				return fmt.Sprintf("hsl(%d, 90%%, 45%%)", 60+int(float64(percentage-50)*1.2))
			}
		},
		"getCoverageColorAlpha": func(percentage int) string {
			// Color gradient with alpha: red (0%) -> yellow (50%) -> green (100%)
			if percentage < 50 {
				// Red to yellow (0-50%)
				return fmt.Sprintf("hsla(%d, 90%%, 50%%, 0.15)", int(float64(percentage)*1.2))
			} else {
				// Yellow to green (50-100%)
				return fmt.Sprintf("hsla(%d, 90%%, 45%%, 0.15)", 60+int(float64(percentage-50)*1.2))
			}
		},
	}

	// Parse our HTML template
	tmpl, err := template.New("coverage_report.html").Funcs(functionMap).Parse(string(htmlReportTemplate))
	if err != nil {
		return "", fmt.Errorf("could not export report, failed to parse report template: %v", err)
	}

	// If the directory doesn't exist, create it.
	err = utils.MakeDirectory(reportDir)
	if err != nil {
		return "", err
	}

	// Create our report file
	htmlReportPath := filepath.Join(reportDir, "coverage_report.html")
	file, err := os.Create(htmlReportPath)
	if err != nil {
		_ = file.Close()
		return "", fmt.Errorf("could not export report, failed to open file for writing: %v", err)
	}

	// Execute the template and write it back to file.
	err = tmpl.Execute(file, sourceAnalysis)
	fileCloseErr := file.Close()
	if err == nil {
		err = fileCloseErr
	}
	return htmlReportPath, err
}

// WriteLCOVReport takes a previously performed source analysis and generates an LCOV report from it.
func WriteLCOVReport(sourceAnalysis *SourceAnalysis, reportDir string) (string, error) {
	// Generate the LCOV report.
	lcovReport := sourceAnalysis.GenerateLCOVReport()

	// If the directory doesn't exist, create it.
	err := utils.MakeDirectory(reportDir)
	if err != nil {
		return "", err
	}

	// Write the LCOV report to a file.
	lcovReportPath := filepath.Join(reportDir, "lcov.info")
	err = os.WriteFile(lcovReportPath, []byte(lcovReport), 0644)
	if err != nil {
		return "", fmt.Errorf("could not export LCOV report: %v", err)
	}

	return lcovReportPath, nil
}
