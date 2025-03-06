package coverage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/crytic/medusa/compilation/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"golang.org/x/exp/maps"
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

// GenerateLCOVReport generates an LCOV report from the source analysis.
// The spec of the format is here https://github.com/linux-test-project/lcov/blob/07a1127c2b4390abf4a516e9763fb28a956a9ce4/man/geninfo.1#L989
func (s *SourceAnalysis) GenerateLCOVReport() string {
	var linesHit, linesInstrumented int
	var buffer bytes.Buffer
	buffer.WriteString("TN:\n")
	for _, file := range s.SortedFiles() {
		// SF:<path to the source file>
		buffer.WriteString(fmt.Sprintf("SF:%s\n", file.Path))
		for idx, line := range file.Lines {
			if line.IsActive {
				// DA:<line number>,<execution count>
				if line.IsCovered {
					buffer.WriteString(fmt.Sprintf("DA:%d,%d\n", idx+1, line.SuccessHitCount))
					linesHit++
				} else {
					buffer.WriteString(fmt.Sprintf("DA:%d,%d\n", idx+1, 0))
				}
				linesInstrumented++
			}
		}
		// FN:<line number>,<function name>
		// FNDA:<execution count>,<function name>
		for _, fn := range file.Functions {
			byteStart := types.GetSrcMapStart(fn.Src)
			length := types.GetSrcMapLength(fn.Src)

			startLine := sort.Search(len(file.CumulativeOffsetByLine), func(i int) bool {
				return file.CumulativeOffsetByLine[i] > byteStart
			})
			endLine := sort.Search(len(file.CumulativeOffsetByLine), func(i int) bool {
				return file.CumulativeOffsetByLine[i] > byteStart+length
			})

			// We are treating any line hit in the definition as a hit for the function.
			hit := 0
			for i := startLine; i < endLine; i++ {
				// index iz zero based, line numbers are 1 based
				if file.Lines[i-1].IsActive && file.Lines[i-1].IsCovered {
					hit = 1
				}

			}

			// TODO: handle fallback, receive, and constructor
			if fn.Name != "" {
				buffer.WriteString(fmt.Sprintf("FN:%d,%s\n", startLine, fn.Name))
				buffer.WriteString(fmt.Sprintf("FNDA:%d,%s\n", hit, fn.Name))
			}

		}
		buffer.WriteString("end_of_record\n")
	}

	return buffer.String()
}

// SourceFileAnalysis describes coverage information for a given source file.
type SourceFileAnalysis struct {
	// Path describes the file path of the source file. This is kept here for access during report generation.
	Path string

	// CumulativeOffsetByLine describes the cumulative byte offset for each line in the source file.
	// For example, for a file with 5 lines, the list might look like: [0, 45, 98, 132, 189], where each number is the byte offset of the line's starting position
	// This allows us to quickly determine which line a given byte offset falls within using a binary search.
	CumulativeOffsetByLine []int

	// Lines describes information about a given source line and its coverage.
	Lines []*SourceLineAnalysis

	// Functions is a list of functions defined in the source file
	Functions []*types.FunctionDefinition
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

	// SuccessHitCount describes how many times this line was executed successfully
	SuccessHitCount uint64

	// RevertHitCount describes how many times this line reverted during execution
	RevertHitCount uint64

	// IsCoveredReverted indicates whether the source line has been executed before reverting.
	IsCoveredReverted bool
}

// GetUniquePCsCount returns the number of PCs in all contracts hit by our tests.
func GetUniquePCsCount(compilations []types.Compilation, coverageMaps *CoverageMaps) (int, error) {
	uniquePCs := 0

	// Loop through all sources in all compilations to process coverage information.
	for _, compilation := range compilations {
		for _, source := range compilation.SourcePathToArtifact {
			// Loop for each contract in this source
			for _, contract := range source.Contracts {
				// Skip interfaces.
				if contract.Kind == types.ContractKindInterface {
					continue
				}
				// Obtain coverage map data for this contract.
				initCoverageMapData, err := coverageMaps.GetContractCoverageMap(contract.InitBytecode, true)
				if err != nil {
					return 0, fmt.Errorf("could not perform source code analysis due to error fetching init coverage map data: %v", err)
				}
				runtimeCoverageMapData, err := coverageMaps.GetContractCoverageMap(contract.RuntimeBytecode, false)
				if err != nil {
					return 0, fmt.Errorf("could not perform source code analysis due to error fetching runtime coverage map data: %v", err)
				}

				coverageMaps.updateLock.Lock()
				uniquePCs += getContractPCsHit(contract.InitBytecode, initCoverageMapData)
				uniquePCs += getContractPCsHit(contract.RuntimeBytecode, runtimeCoverageMapData)
				coverageMaps.updateLock.Unlock()
			}
		}
	}
	return uniquePCs, nil
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
		for sourcePath := range compilation.SourcePathToArtifact {
			// If we have no source code loaded for this source, skip it.
			if _, ok := compilation.SourceCode[sourcePath]; !ok {
				return nil, fmt.Errorf("could not perform source code analysis, code was not cached for '%v'", sourcePath)
			}

			lines, cumulativeOffset := parseSourceLines(compilation.SourceCode[sourcePath])
			funcs := make([]*types.FunctionDefinition, 0)

			var ast types.AST
			b, err := json.Marshal(compilation.SourcePathToArtifact[sourcePath].Ast)
			if err != nil {
				return nil, fmt.Errorf("could not encode AST from sources: %v", err)
			}
			err = json.Unmarshal(b, &ast)
			if err != nil {
				return nil, fmt.Errorf("could not parse AST from sources: %v", err)
			}

			for _, node := range ast.Nodes {

				if node.GetNodeType() == "FunctionDefinition" {
					fn := node.(types.FunctionDefinition)
					funcs = append(funcs, &fn)
				}
				if node.GetNodeType() == "ContractDefinition" {
					contract := node.(types.ContractDefinition)
					if contract.Kind == types.ContractKindInterface {
						continue
					}
					for _, subNode := range contract.Nodes {
						if subNode.GetNodeType() == "FunctionDefinition" {
							fn := subNode.(types.FunctionDefinition)
							funcs = append(funcs, &fn)
						}
					}
				}

			}

			// Obtain the parsed source code lines for this source.
			if _, ok := sourceAnalysis.Files[sourcePath]; !ok {
				sourceAnalysis.Files[sourcePath] = &SourceFileAnalysis{
					Path:                   sourcePath,
					CumulativeOffsetByLine: cumulativeOffset,
					Lines:                  lines,
					Functions:              funcs,
				}
			}

		}
	}

	// Loop through all sources in all compilations to process coverage information.
	for _, compilation := range compilations {
		for _, source := range compilation.SourcePathToArtifact {
			// Loop for each contract in this source
			for _, contract := range source.Contracts {
				// Skip interfaces.
				if contract.Kind == types.ContractKindInterface {
					continue
				}
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

				// Filter our source maps
				initSourceMap = filterSourceMaps(compilation, initSourceMap)
				runtimeSourceMap = filterSourceMaps(compilation, runtimeSourceMap)

				// Analyze both init and runtime coverage for our source lines.
				err = analyzeContractSourceCoverage(compilation, sourceAnalysis, initSourceMap, contract.InitBytecode, initCoverageMapData)
				if err != nil {
					return nil, err
				}
				err = analyzeContractSourceCoverage(compilation, sourceAnalysis, runtimeSourceMap, contract.RuntimeBytecode, runtimeCoverageMapData)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return sourceAnalysis, nil
}

// getContractPCsHit returns the number of PCs in this contract hit by our tests.
func getContractPCsHit(bytecode []byte, contractCoverageData *ContractCoverageMap) int {
	if len(bytecode) == 0 || contractCoverageData == nil {
		return 0
	}
	succHitCounts, revertHitCounts := determineLinesCovered(contractCoverageData, bytecode)
	if succHitCounts == nil || revertHitCounts == nil {
		return 0
	}
	pcsHit := 0
	for i, ct := range succHitCounts {
		if ct > 0 || revertHitCounts[i] > 0 {
			pcsHit++
		}
	}
	return pcsHit
}

// analyzeContractSourceCoverage takes a compilation, a SourceAnalysis, the source map they were derived from,
// a lookup of instruction index->offset, and coverage map data. It updates the coverage source line mapping with
// coverage data, after analyzing the coverage data for the given file in the given compilation.
// Returns an error if one occurs.
func analyzeContractSourceCoverage(compilation types.Compilation, sourceAnalysis *SourceAnalysis, sourceMap types.SourceMap, bytecode []byte, contractCoverageData *ContractCoverageMap) error {
	var succHitCounts, revertHitCounts []uint64
	if len(bytecode) > 0 && contractCoverageData != nil {
		succHitCounts, revertHitCounts = determineLinesCovered(contractCoverageData, bytecode)
	} else { // Probably because we didn't hit this contract at all...
		succHitCounts = nil
		revertHitCounts = nil
	}

	// Loop through each source map element
	for _, sourceMapElement := range sourceMap {
		// If this source map element doesn't map to any file (compiler generated inline code), it will have no
		// relevance to the coverage map, so we skip it.
		if sourceMapElement.SourceUnitID == -1 {
			continue
		}

		// Obtain our source for this file ID
		sourcePath, idExists := compilation.SourceIdToPath[sourceMapElement.SourceUnitID]

		// TODO: We may also go out of bounds because this maps to a "generated source" which we do not have.
		//  For now, we silently skip these cases.
		if !idExists {
			continue
		}

		// Capture the hit count of the source map element.
		var succHitCount, revertHitCount uint64
		if succHitCounts != nil {
			succHitCount = succHitCounts[sourceMapElement.Index]
		} else {
			succHitCount = 0
		}
		if revertHitCounts != nil {
			revertHitCount = revertHitCounts[sourceMapElement.Index]
		} else {
			revertHitCount = 0
		}

		// Obtain the source file this element maps to.
		if sourceFile, ok := sourceAnalysis.Files[sourcePath]; ok {
			// Mark all lines which fall within this range.
			start := sourceMapElement.Offset

			startLine := sort.Search(len(sourceFile.CumulativeOffsetByLine), func(i int) bool {
				return sourceFile.CumulativeOffsetByLine[i] > start
			})

			// index iz zero based, line numbers are 1 based
			sourceLine := sourceFile.Lines[startLine-1]

			// Check if the line is within range
			if sourceMapElement.Offset < sourceLine.End {
				// Mark the line active/executable.
				sourceLine.IsActive = true

				// Set its coverage state and increment hit counts
				if succHitCount > sourceLine.SuccessHitCount {
					// We do max rather than += because if we did += then lines with multiple instructions
					// would be weighted higher than lines with single instructions
					sourceLine.SuccessHitCount = succHitCount
				}
				sourceLine.RevertHitCount += revertHitCount // On the other hand, reverts from multiple instructions add as expected
				sourceLine.IsCovered = sourceLine.IsCovered || sourceLine.SuccessHitCount > 0
				sourceLine.IsCoveredReverted = sourceLine.IsCoveredReverted || sourceLine.RevertHitCount > 0

			}
		} else {
			return fmt.Errorf("could not perform source code analysis, missing source '%v'", sourcePath)
		}

	}
	return nil
}

// determineLinesCovered takes a ContractCoverageMap and a contract's bytecode, and determines which program counters were hit.
// Returns two slices: one for successful hits and one for reverts. These slices are indexed by instruction index (not program counter),
// and their values are the number of hits (0 if the PC was not hit).
func determineLinesCovered(cm *ContractCoverageMap, bytecode []byte) ([]uint64, []uint64) {
	indexToOffset := getInstructionIndexToOffsetLookup(bytecode)

	// executedMakers as src -> dst -> hit count, and dst -> src -> hit count
	execMarkersSrcDst, execMarkersDstSrc := getExecMarkersMapping(cm.executedMarkers)

	successfulHits := make([]uint64, len(indexToOffset))
	revertedHits := make([]uint64, len(indexToOffset))

	// Traverse the instructions from top to bottom, keeping track of hit count as we go
	hit := uint64(0)
	for idx, pc := range indexToOffset {
		enterCount := uint64(0)    // count of jumpdest + contract initial enter (ENTER_MARKER_XOR)
		revertCount := uint64(0)   // count of revert (REVERT_MARKER_XOR)
		allLeaveCount := uint64(0) // count of jump + return (RETURN_MAKRER_XOR) + revert (REVERT_MARKER_XOR)

		for _, hitHere := range execMarkersDstSrc[uint64(pc)] {
			enterCount += hitHere
		}
		revertCount = execMarkersSrcDst[uint64(pc)][REVERT_MARKER_XOR]
		for _, hitHere := range execMarkersSrcDst[uint64(pc)] {
			allLeaveCount += hitHere
		}

		// Test some conditions that should always hold...
		op := vm.OpCode(bytecode[pc])                                                         // Used only for checks below
		isJumpOrReturn := op == vm.JUMP || op == vm.JUMPI || op == vm.RETURN || op == vm.STOP // Used only for checks below
		if hit+enterCount < hit {
			fmt.Printf("WARNING: Overflow while generating coverage report, during `hit += enterCount` calculation. The coverage report will be inaccurate. This is a bug; please report it at https://github.com/crytic/medusa/issues. Debug info: hit: %d, enterCount: %d, revertCount: %d, allLeaveCount: %d, idx: %d, pc: %d, op: %d, isJumpOrReturn: %t, len(bytecode): %d, len(indexToOffset): %d.\n", hit, enterCount, revertCount, allLeaveCount, idx, pc, op, isJumpOrReturn, len(bytecode), len(indexToOffset))
		}
		if hit+enterCount < revertCount {
			fmt.Printf("WARNING: Underflow while generating coverage report, during `hit - revertCount` calculation. The coverage report will be inaccurate. This is a bug; please report it at https://github.com/crytic/medusa/issues. Debug info: hit: %d, enterCount: %d, revertCount: %d, allLeaveCount: %d, idx: %d, pc: %d, op: %d, isJumpOrReturn: %t, len(bytecode): %d, len(indexToOffset): %d.\n", hit, enterCount, revertCount, allLeaveCount, idx, pc, op, isJumpOrReturn, len(bytecode), len(indexToOffset))
		}
		if hit+enterCount < allLeaveCount {
			fmt.Printf("WARNING: Underflow while generating coverage report, during `hit -= allLeaveCount` calculation. The coverage report will be inaccurate. This is a bug; please report it at https://github.com/crytic/medusa/issues. Debug info: hit: %d, enterCount: %d, revertCount: %d, allLeaveCount: %d, idx: %d, pc: %d, op: %d, isJumpOrReturn: %t, len(bytecode): %d, len(indexToOffset): %d.\n", hit, enterCount, revertCount, allLeaveCount, idx, pc, op, isJumpOrReturn, len(bytecode), len(indexToOffset))
		}
		if isJumpOrReturn && hit+enterCount != allLeaveCount {
			fmt.Printf("WARNING: Unexpected condition while generating coverage report: return or jump does not reset hit count to 0. The coverage report will be inaccurate. This is a bug; please report it at https://github.com/crytic/medusa/issues. Debug info: hit: %d, enterCount: %d, revertCount: %d, allLeaveCount: %d, idx: %d, pc: %d, op: %d, isJumpOrReturn: %t, len(bytecode): %d, len(indexToOffset): %d.\n", hit, enterCount, revertCount, allLeaveCount, idx, pc, op, isJumpOrReturn, len(bytecode), len(indexToOffset))
		}
		if allLeaveCount-revertCount > 0 && hit+enterCount != allLeaveCount {
			// The check is allLeaveCount-revertCount > 0 rather than just allLeaveCount > 0 since reverts don't have to reset hit to 0
			fmt.Printf("WARNING: Unexpected condition while generating coverage report: positive allLeaveCount does not reset hit count to 0. The coverage report will be inaccurate. This is a bug; please report it at https://github.com/crytic/medusa/issues. Debug info: hit: %d, enterCount: %d, revertCount: %d, allLeaveCount: %d, idx: %d, pc: %d, op: %d, isJumpOrReturn: %t, len(bytecode): %d, len(indexToOffset): %d.\n", hit, enterCount, revertCount, allLeaveCount, idx, pc, op, isJumpOrReturn, len(bytecode), len(indexToOffset))
		}

		// Modify hit based on coverage for this line, and record results
		hit += enterCount
		successfulHits[idx] = hit - revertCount
		revertedHits[idx] = revertCount
		hit -= allLeaveCount
	}
	if hit != 0 {
		fmt.Printf("WARNING: Nonzero final hit count. The coverage report will be inaccurate. This is a bug; please report it at https://github.com/crytic/medusa/issues. Debug info: hit: %d, len(bytecode): %d, len(indexToOffset): %d.\n", hit, len(bytecode), len(indexToOffset))
	}

	return successfulHits, revertedHits
}

// GetInstructionIndexToOffsetLookup obtains a slice where each index of the slice corresponds to an instruction index,
// and the element of the slice represents the instruction offset.
func getInstructionIndexToOffsetLookup(bytecode []byte) []int {
	// Create our resulting lookup
	indexToOffsetLookup := make([]int, 0, len(bytecode))

	// Loop through all byte code
	currentOffset := 0
	for currentOffset < len(bytecode) {
		// Obtain the indexed instruction and add the current offset to our lookup at this index.
		op := vm.OpCode(bytecode[currentOffset])
		indexToOffsetLookup = append(indexToOffsetLookup, currentOffset)

		// Next, calculate the length of data that follows this instruction.
		operandCount := 0
		if op.IsPush() {
			if op == vm.PUSH0 {
				operandCount = 0
			} else {
				operandCount = int(op) - int(vm.PUSH1) + 1
			}
		}

		// Advance the offset past this instruction and its operands.
		currentOffset += operandCount + 1
	}
	return indexToOffsetLookup
}

// getExecMarkersMapping takes a set of executed markers, and sorts it into a map from src -> dst -> hit count, and a map of dst -> src -> hit count.
// This is a helper function used by determineLinesCovered.
func getExecMarkersMapping(execMarkers map[uint64]uint64) (map[uint64]map[uint64]uint64, map[uint64]map[uint64]uint64) {
	execMarkersSrcDst := make(map[uint64]map[uint64]uint64)
	execMarkersDstSrc := make(map[uint64]map[uint64]uint64)

	for marker, hitCount := range execMarkers {
		// Lower, upper 32 bits
		dst := marker & 0xFFFFFFFF
		src := marker >> 32
		if _, ok := execMarkersSrcDst[src]; !ok {
			execMarkersSrcDst[src] = make(map[uint64]uint64, 1)
		}
		if _, ok := execMarkersDstSrc[dst]; !ok {
			execMarkersDstSrc[dst] = make(map[uint64]uint64, 1)
		}
		execMarkersSrcDst[src][dst] = hitCount
		execMarkersDstSrc[dst][src] = hitCount
	}

	return execMarkersSrcDst, execMarkersDstSrc
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
		if _, exists := compilation.SourceIdToPath[sourceMapElement.SourceUnitID]; !exists {
			// TODO: We may also go out of bounds because this maps to a "generated source" which we do not have.
			//  For now, we silently skip these cases.
			continue
		}

		// Verify this source map does not overlap another
		encapsulatesOtherMapping := false
		for x, sourceMapElement2 := range sourceMap {
			if i != x && sourceMapElement.SourceUnitID == sourceMapElement2.SourceUnitID &&
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
func parseSourceLines(sourceCode []byte) ([]*SourceLineAnalysis, []int) {
	// Create our lines and a variable to track where our current line start offset is.
	var lines []*SourceLineAnalysis
	var lineStart int
	var cumulativeOffset []int

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
		cumulativeOffset = append(cumulativeOffset, int(lineStart))
		lineStart = lineEnd
	}

	// Return the resulting lines
	return lines, cumulativeOffset
}
