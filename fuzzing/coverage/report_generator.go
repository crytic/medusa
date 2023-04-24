package coverage

import (
	_ "embed"
	"fmt"
	"github.com/crytic/medusa/compilation/types"
	"html/template"
	"os"
	"time"
)

var (
	//go:embed report_template.gohtml
	htmlReportTemplate []byte
)

// GenerateReport takes a set of CoverageMaps and compilations, and produces a coverage report using them, detailing
// all source mapped ranges of the source files which were covered or not.
// Returns an error if one occurred.
func GenerateReport(coverageMaps *CoverageMaps, compilations []types.Compilation) error {
	// Create a map of source file paths to coverage lines.
	sourceLinesByFile := make(map[string][]*coverageSourceLine)

	// The fuzzer generates coverage per address and codehash (e.g map[address] -> map[codeHash] -> coverage)
	// so for each codeHash we have to figure out to which contract definition it belongs to.
	// first we iterate over all the contract definitions and calculate the codeHashes then store them in a mapping
	for _, compilation := range compilations {
		for sourcePath, source := range compilation.Sources {
			// If we have no source code loaded for this source, skip it.
			if _, ok := compilation.SourceCode[sourcePath]; !ok {
				return fmt.Errorf("could not generate coverage report as source code was not cached for source '%v'", sourcePath)
			}

			// Obtain the parsed source code lines for this source.
			if _, ok := sourceLinesByFile[sourcePath]; !ok {
				sourceLinesByFile[sourcePath] = splitSourceCode(compilation.SourceCode[sourcePath])
			}

			// Loop for each contract in this source
			for _, contract := range source.Contracts {
				// Obtain coverage map data for this contract.
				var (
					initCoverageMapBytecodeData    *CoverageMapBytecodeData
					runtimeCoverageBytecodeMapData *CoverageMapBytecodeData
				)
				initCoverageMapData, err := coverageMaps.GetContractCoverageMap(contract.InitBytecode)
				if err != nil {
					return fmt.Errorf("could not generate coverage report due to error when obtaining init coverage map data: %v", err)
				}
				if initCoverageMapData != nil {
					initCoverageMapBytecodeData = initCoverageMapData.initBytecodeCoverage
				}
				runtimeCoverageMapData, err := coverageMaps.GetContractCoverageMap(contract.RuntimeBytecode)
				if err != nil {
					return fmt.Errorf("could not generate coverage report due to error when obtaining runtime coverage map data: %v", err)
				}
				if runtimeCoverageMapData != nil {
					runtimeCoverageBytecodeMapData = runtimeCoverageMapData.deployedBytecodeCoverage
				}

				// Parse the source map for this contract.
				initSourceMap, err := types.ParseSourceMap(contract.SrcMapsInit)
				if err != nil {
					return fmt.Errorf("could not generate coverage report due to error when obtaining init source map: %v", err)
				}
				runtimeSourceMap, err := types.ParseSourceMap(contract.SrcMapsRuntime)
				if err != nil {
					return fmt.Errorf("could not generate coverage report due to error when obtaining runtime source map: %v", err)
				}

				// Parse our instruction index to offset lookups
				initInstructionOffsetLookup, err := initSourceMap.GetInstructionIndexToOffsetLookup(contract.InitBytecode)
				if err != nil {
					return fmt.Errorf("could not generate coverage report due to error when parsing init byte code: %v", err)
				}
				runtimeInstructionOffsetLookup, err := runtimeSourceMap.GetInstructionIndexToOffsetLookup(contract.RuntimeBytecode)
				if err != nil {
					return fmt.Errorf("could not generate coverage report due to error when parsing runtime byte code: %v", err)
				}

				// Filter our source maps
				initSourceMap = filterSourceMaps(compilation, initSourceMap)
				runtimeSourceMap = filterSourceMaps(compilation, runtimeSourceMap)

				// Analyze both init and runtime coverage for our source lines.
				err = analyzeReportCoverageMapData(compilation, sourceLinesByFile, initSourceMap, initInstructionOffsetLookup, initCoverageMapBytecodeData)
				if err != nil {
					return err
				}
				err = analyzeReportCoverageMapData(compilation, sourceLinesByFile, runtimeSourceMap, runtimeInstructionOffsetLookup, runtimeCoverageBytecodeMapData)
				if err != nil {
					return err
				}
			}
		}
	}

	// Finally, export the report data we analyzed.
	// TODO: Replace this static path with one that is derived from config
	outputPath := fmt.Sprintf("C:\\Users\\X\\Documents\\___\\report_%v.html", time.Now().UnixNano())
	return exportCoverageReport(sourceLinesByFile, outputPath)
}

// filterSourceMaps takes a given source map and filters it so overlapping (superset) source map elements are filtered
// out, in addition to any which do not map to any source code. This is necessary as some source map entries select
// an entire method definition.
// Returns the filtered source map.
func filterSourceMaps(compilation types.Compilation, sourceMap types.SourceMap) types.SourceMap {
	// Create our resulting source map
	filteredMap := make(types.SourceMap, 0)

	// Loop for each source map entry and determine if it should be included.
	for i, sourceMapElement := range sourceMap {
		// Verify this file ID is not out of bounds for a source file index
		if sourceMapElement.FileID < 0 || sourceMapElement.FileID >= len(compilation.SourceList) {
			// TODO: We may also go out of bounds because this maps to a "generated source" which we do not have.
			//  For now, we silently skip these cases.
			continue
		}

		// Verify this source map does not overlap another
		encapsulatesOtherMapping := false
		for x, sourceMapElement2 := range sourceMap {
			if i != x && sourceMapElement.FileID == sourceMapElement2.FileID &&
				!(sourceMapElement.Offset == sourceMapElement2.Offset && sourceMapElement.Length == sourceMapElement2.Length) {
				if sourceMapElement2.Offset >= sourceMapElement.Offset &&
					sourceMapElement2.Offset+sourceMapElement2.Length <= sourceMapElement.Offset+sourceMapElement.Length {
					encapsulatesOtherMapping = true
					break
				}
			}
		}

		if !encapsulatesOtherMapping {
			filteredMap = append(filteredMap, sourceMapElement)
		}
	}
	return filteredMap
}

func analyzeReportCoverageMapData(compilation types.Compilation, sourceLinesByFile map[string][]*coverageSourceLine, sourceMap types.SourceMap, instructionOffsetLookup []int, coverageMapData *CoverageMapBytecodeData) error {
	// Loop through each source map element
	for _, sourceMapElement := range sourceMap {
		// If this source map element doesn't map to any file (compiler generated inline code), it will have no
		// relevance to the coverage map, so we skip it.
		if sourceMapElement.FileID == -1 {
			continue
		}

		// Verify this file ID is not out of bounds for a source file index
		if sourceMapElement.FileID < 0 || sourceMapElement.FileID >= len(compilation.SourceList) {
			// TODO: We may also go out of bounds because this maps to a "generated source" which we do not have.
			//  For now, we silently skip these cases.
			continue
		}

		// Obtain our source for this file ID
		sourcePath := compilation.SourceList[sourceMapElement.FileID]

		// Check if the source map element was executed.
		sourceMapElementCovered := coverageMapData.isCovered(instructionOffsetLookup[sourceMapElement.Index])

		// Obtain the source file this element maps to.
		if sourceLines, ok := sourceLinesByFile[sourcePath]; ok {
			// Mark all lines which fall within this range.
			matchedSourceLine := false
			for _, sourceLine := range sourceLines {
				// Check if the line is within range
				if sourceMapElement.Offset >= sourceLine.Start && sourceMapElement.Offset < sourceLine.End {
					// Mark the line active/executable.
					sourceLine.IsActive = true

					// Set its coverage state
					sourceLine.IsCovered = sourceMapElementCovered

					// Indicate we matched a source line, so when we stop matching sequentially, we know we can exit
					// early.
					matchedSourceLine = true
				} else if matchedSourceLine {
					break
				}
			}
		} else {
			return fmt.Errorf("could not generate report, failed to resolve source lines by source path")
		}

	}
	return nil
}

func exportCoverageReport(sourceLinesByFile map[string][]*coverageSourceLine, outputPath string) error {
	// Define template compatible structures.
	type SourceFile struct {
		Path  string
		Lines []*coverageSourceLine
	}

	// Convert all files to the template friendly format.
	sourceFiles := make([]SourceFile, 0)
	for filePath, lines := range sourceLinesByFile {
		// Add our source file with the lines.
		sourceFiles = append(sourceFiles, SourceFile{
			Path:  filePath,
			Lines: lines,
		})
	}
	// Parse our HTML template
	tmpl, err := template.New("coverage_report.html").Parse(string(htmlReportTemplate))
	if err != nil {
		return fmt.Errorf("could not export report, failed to parse report template: %v", err)
	}

	// Create our report file
	file, err := os.Create(outputPath)
	if err != nil {
		// TODO error handling
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
