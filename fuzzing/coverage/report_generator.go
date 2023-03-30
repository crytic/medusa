package coverage

import (
	"bytes"
	"fmt"
	"github.com/trailofbits/medusa/compilation/types"
)

// GenerateReport takes a set of CoverageMaps and compilations, and produces a coverage report using them, detailing
// all source mapped ranges of the source files which were covered or not.
// Returns an error if one occurred.
func GenerateReport(coverageMaps *CoverageMaps, compilations []types.Compilation) error {
	// Create a map of source file paths to coverage lines.
	sourceLinesByFile := make(map[string][]coverageSourceLine)

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
			sourceLines := sourceLinesByFile[sourcePath]

			// Loop for each contract in this source
			for contractName, contract := range source.Contracts {
				// Obtain coverage map data for this contract.
				initCoverageMapData, err := coverageMaps.GetCoverageMapData(contract.InitBytecode)
				if err != nil {
					return fmt.Errorf("could not generate coverage report due to error when obtaining init coverage map data: %v", err)
				}
				runtimeCoverageMapData, err := coverageMaps.GetCoverageMapData(contract.RuntimeBytecode)
				if err != nil {
					return fmt.Errorf("could not generate coverage report due to error when obtaining runtime coverage map data: %v", err)
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

				_, _, _, _, _, _ = sourceLines, contractName, initCoverageMapData, runtimeCoverageMapData, initInstructionOffsetLookup, runtimeInstructionOffsetLookup
			}
		}
	}
	return nil
}

// coverageSourceLine indicates whether the
type coverageSourceLine struct {
	IsActive bool
	Start    int
	End      int
	Contents []byte
	Covered  bool
}

// splitSourceCode splits the provided source code into coverageSourceLine objects.
// Returns the coverageSourceLine objects.
func splitSourceCode(sourceCode []byte) []coverageSourceLine {
	// Create our lines and a variable to track where our current line start offset is.
	var lines []coverageSourceLine
	var lineStart int

	// Split the source code on new line characters
	sourceCodeLinesBytes := bytes.Split(sourceCode, []byte("\n"))

	// For each source code line, initialize a struct that defines its start/end offsets, set its contents.
	for i := 0; i < len(sourceCodeLinesBytes); i++ {
		lineEnd := lineStart + len(sourceCodeLinesBytes[i]) + 1
		lines = append(lines, coverageSourceLine{
			IsActive: false,
			Start:    lineStart,
			End:      lineEnd,
			Contents: sourceCodeLinesBytes[i],
			Covered:  false,
		})
		lineStart = lineEnd
	}

	// Return the resulting lines
	return lines
}
