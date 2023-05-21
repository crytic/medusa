package coverage

import (
	"bytes"
	"fmt"
	"github.com/crytic/medusa/compilation/types"
	"golang.org/x/exp/maps"
	"sort"
)

// SourceAnalysis describes source code coverage across a list of compilations, after analyzing associated CoverageMaps.
type SourceAnalysis struct {
	// Files describes the analysis results for a given source file path.
	Files map[string]*SourceFileAnalysis
}

// SortedFiles returns a list of Files within the SourceAnalysis, sorted by source file path in alphabetical order.
func (s *SourceAnalysis) SortedFiles() []*SourceFileAnalysis {
	// Copy all source files from our analysis into a list.
	sourceFiles := maps.Values(s.Files)

	// Sort source files by path
	sort.Slice(sourceFiles, func(x, y int) bool {
		return sourceFiles[x].Path < sourceFiles[y].Path
	})

	return sourceFiles
}

// LineCount returns the count of lines across all source files.
func (s *SourceAnalysis) LineCount() int {
	count := 0
	for _, file := range s.Files {
		count += len(file.Lines)
	}
	return count
}

// ActiveLineCount returns the count of lines that are marked executable/active across all source files.
func (s *SourceAnalysis) ActiveLineCount() int {
	count := 0
	for _, file := range s.Files {
		count += file.ActiveLineCount()
	}
	return count
}

// CoveredLineCount returns the count of lines that were covered across all source files.
func (s *SourceAnalysis) CoveredLineCount() int {
	count := 0
	for _, file := range s.Files {
		count += file.CoveredLineCount()
	}
	return count
}

// SourceFileAnalysis describes coverage information for a given source file.
type SourceFileAnalysis struct {
	// Path describes the file path of the source file. This is kept here for access during report generation.
	Path string

	// Lines describes information about a given source line and its coverage.
	Lines []*SourceLineAnalysis
}

// ActiveLineCount returns the count of lines that are marked executable/active within the source file.
func (s *SourceFileAnalysis) ActiveLineCount() int {
	count := 0
	for _, line := range s.Lines {
		if line.IsActive {
			count++
		}
	}
	return count
}

// CoveredLineCount returns the count of lines that were covered within the source file.
func (s *SourceFileAnalysis) CoveredLineCount() int {
	count := 0
	for _, line := range s.Lines {
		if line.IsCovered || line.IsCoveredReverted {
			count++
		}
	}
	return count
}

// SourceLineAnalysis describes coverage information for a specific source file line.
type SourceLineAnalysis struct {
	// IsActive indicates the given source line was executable.
	IsActive bool

	// Start describes the starting byte offset of the line in its parent source file.
	Start int

	// End describes the ending byte offset of the line in its parent source file.
	End int

	// Contents describe the bytes associated with the given source line.
	Contents []byte

	// IsCovered indicates whether the source line has been executed without reverting.
	IsCovered bool

	// IsCoveredReverted indicates whether the source line has been executed before reverting.
	IsCoveredReverted bool
}

// AnalyzeSourceCoverage takes a list of compilations and a set of coverage maps, and performs source analysis
// to determine source coverage information.
// Returns a SourceAnalysis object, or an error if one occurs.
func AnalyzeSourceCoverage(compilations []types.Compilation, coverageMaps *CoverageMaps) (*SourceAnalysis, error) {
	// Create a new source analysis object
	sourceAnalysis := &SourceAnalysis{
		Files: make(map[string]*SourceFileAnalysis),
	}

	// Loop through all sources in all compilations to add them to our source file analysis container.
	for _, compilation := range compilations {
		for sourcePath := range compilation.Sources {
			// If we have no source code loaded for this source, skip it.
			if _, ok := compilation.SourceCode[sourcePath]; !ok {
				return nil, fmt.Errorf("could not perform source code analysis, code was not cached for '%v'", sourcePath)
			}

			// Obtain the parsed source code lines for this source.
			if _, ok := sourceAnalysis.Files[sourcePath]; !ok {
				sourceAnalysis.Files[sourcePath] = &SourceFileAnalysis{
					Path:  sourcePath,
					Lines: parseSourceLines(compilation.SourceCode[sourcePath]),
				}
			}
		}
	}

	// Loop through all sources in all compilations to process coverage information.
	for _, compilation := range compilations {
		for _, source := range compilation.Sources {
			// Loop for each contract in this source
			for _, contract := range source.Contracts {
				// Obtain coverage map data for this contract.
				initCoverageMapData, err := coverageMaps.GetContractCoverageMap(contract.InitBytecode, true)
				if err != nil {
					return nil, fmt.Errorf("could not perform source code analysis due to error fetching init coverage map data: %v", err)
				}
				runtimeCoverageMapData, err := coverageMaps.GetContractCoverageMap(contract.RuntimeBytecode, false)
				if err != nil {
					return nil, fmt.Errorf("could not perform source code analysis due to error fetching runtime coverage map data: %v", err)
				}

				// Parse the source map for this contract.
				initSourceMap, err := types.ParseSourceMap(contract.SrcMapsInit)
				if err != nil {
					return nil, fmt.Errorf("could not perform source code analysis due to error fetching init source map: %v", err)
				}
				runtimeSourceMap, err := types.ParseSourceMap(contract.SrcMapsRuntime)
				if err != nil {
					return nil, fmt.Errorf("could not perform source code analysis due to error fetching runtime source map: %v", err)
				}

				// Parse our instruction index to offset lookups
				initInstructionOffsetLookup, err := initSourceMap.GetInstructionIndexToOffsetLookup(contract.InitBytecode)
				if err != nil {
					return nil, fmt.Errorf("could not perform source code analysis due to error parsing init byte code: %v", err)
				}
				runtimeInstructionOffsetLookup, err := runtimeSourceMap.GetInstructionIndexToOffsetLookup(contract.RuntimeBytecode)
				if err != nil {
					return nil, fmt.Errorf("could not perform source code analysis due to error parsing runtime byte code: %v", err)
				}

				// Filter our source maps
				initSourceMap = filterSourceMaps(compilation, initSourceMap)
				runtimeSourceMap = filterSourceMaps(compilation, runtimeSourceMap)

				// Analyze both init and runtime coverage for our source lines.
				err = analyzeContractSourceCoverage(compilation, sourceAnalysis, initSourceMap, initInstructionOffsetLookup, initCoverageMapData)
				if err != nil {
					return nil, err
				}
				err = analyzeContractSourceCoverage(compilation, sourceAnalysis, runtimeSourceMap, runtimeInstructionOffsetLookup, runtimeCoverageMapData)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return sourceAnalysis, nil
}

// analyzeContractSourceCoverage takes a compilation, a SourceAnalysis, the source map they were derived from,
// a lookup of instruction index->offset, and coverage map data. It updates the coverage source line mapping with
// coverage data, after analyzing the coverage data for the given file in the given compilation.
// Returns an error if one occurs.
func analyzeContractSourceCoverage(compilation types.Compilation, sourceAnalysis *SourceAnalysis, sourceMap types.SourceMap, instructionOffsetLookup []int, contractCoverageData *ContractCoverageMap) error {
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
		sourceMapElementCovered := false
		sourceMapElementCoveredReverted := false
		if contractCoverageData != nil {
			sourceMapElementCovered = contractCoverageData.successfulCoverage.IsCovered(instructionOffsetLookup[sourceMapElement.Index])
			sourceMapElementCoveredReverted = contractCoverageData.revertedCoverage.IsCovered(instructionOffsetLookup[sourceMapElement.Index])
		}

		// Obtain the source file this element maps to.
		if sourceFile, ok := sourceAnalysis.Files[sourcePath]; ok {
			// Mark all lines which fall within this range.
			matchedSourceLine := false
			for _, sourceLine := range sourceFile.Lines {
				// Check if the line is within range
				if sourceMapElement.Offset >= sourceLine.Start && sourceMapElement.Offset < sourceLine.End {
					// Mark the line active/executable.
					sourceLine.IsActive = true

					// Set its coverage state
					sourceLine.IsCovered = sourceLine.IsCovered || sourceMapElementCovered
					sourceLine.IsCoveredReverted = sourceLine.IsCoveredReverted || sourceMapElementCoveredReverted

					// Indicate we matched a source line, so when we stop matching sequentially, we know we can exit
					// early.
					matchedSourceLine = true
				} else if matchedSourceLine {
					break
				}
			}
		} else {
			return fmt.Errorf("could not perform source code analysis, missing source '%v'", sourcePath)
		}

	}
	return nil
}

// filterSourceMaps takes a given source map and filters it so overlapping (superset) source map elements are removed.
// In addition to any which do not map to any source code. This is necessary as some source map entries select an
// entire method definition.
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

// parseSourceLines splits the provided source code into SourceLineAnalysis objects.
// Returns the SourceLineAnalysis objects.
func parseSourceLines(sourceCode []byte) []*SourceLineAnalysis {
	// Create our lines and a variable to track where our current line start offset is.
	var lines []*SourceLineAnalysis
	var lineStart int

	// Split the source code on new line characters
	sourceCodeLinesBytes := bytes.Split(sourceCode, []byte("\n"))

	// For each source code line, initialize a struct that defines its start/end offsets, set its contents.
	for i := 0; i < len(sourceCodeLinesBytes); i++ {
		lineEnd := lineStart + len(sourceCodeLinesBytes[i]) + 1
		lines = append(lines, &SourceLineAnalysis{
			IsActive:          false,
			Start:             lineStart,
			End:               lineEnd,
			Contents:          sourceCodeLinesBytes[i],
			IsCovered:         false,
			IsCoveredReverted: false,
		})
		lineStart = lineEnd
	}

	// Return the resulting lines
	return lines
}
