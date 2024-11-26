package coverage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/crytic/medusa/compilation/types"
	"golang.org/x/exp/maps"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/bits"
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
	SuccessHitCount uint

	// RevertHitCount describes how many times this line reverted during execution
	RevertHitCount uint

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

				// // Parse our instruction index to offset lookups
				// initInstructionMarkersLookup, err := initSourceMap.GetInstructionIndexToMarkersLookup(contract.InitBytecode)
				// if err != nil {
				// 	return nil, fmt.Errorf("could not perform source code analysis due to error parsing init byte code: %v", err)
				// }
				// runtimeInstructionMarkersLookup, err := runtimeSourceMap.GetInstructionIndexToMarkersLookup(contract.RuntimeBytecode)
				// if err != nil {
				// 	return nil, fmt.Errorf("could not perform source code analysis due to error parsing runtime byte code: %v", err)
				// }

				// Filter our source maps
				initSourceMap = filterSourceMaps(compilation, initSourceMap)
				runtimeSourceMap = filterSourceMaps(compilation, runtimeSourceMap)

				// Analyze both init and runtime coverage for our source lines.
				err = analyzeContractSourceCoverage(compilation, sourceAnalysis, initSourceMap, /*initInstructionMarkersLookup,*/ contract.InitBytecode, initCoverageMapData)
				if err != nil {
					return nil, err
				}
				err = analyzeContractSourceCoverage(compilation, sourceAnalysis, runtimeSourceMap, /*runtimeInstructionMarkersLookup,*/ contract.RuntimeBytecode, runtimeCoverageMapData)
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
func analyzeContractSourceCoverage(compilation types.Compilation, sourceAnalysis *SourceAnalysis, sourceMap types.SourceMap, /*instructionMarkersLookup map[int][]uint64,*/ bytecode []byte, contractCoverageData *ContractCoverageMap) error {
	if len(bytecode) == 0 { return nil } // TODO
	if contractCoverageData == nil { return nil } // TODO
	succHitCounts, revertHitCounts := determineLinesCovered(contractCoverageData, bytecode)
	if succHitCounts == nil {
		fmt.Println("weird err")
		return nil
	}

	// Loop through each source map element
	for _, sourceMapElement := range sourceMap {
		//fmt.Println(idx, sourceMapElement.Index, len(succHitCounts), len(bytecode))
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
		succHitCount := succHitCounts[sourceMapElement.Index]
		revertHitCount := revertHitCounts[sourceMapElement.Index]
		// TODO this is where we do stuff
		/* if contractCoverageData != nil {
			for _, marker := range instructionMarkersLookup[sourceMapElement.Index] {
				succHitCount += contractCoverageData.successfulCoverage.HitCount(marker) // TODO might overflow on 32 bit
				revertHitCount += contractCoverageData.revertedCoverage.HitCount(marker)
			}
		} */

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
				sourceLine.SuccessHitCount += succHitCount
				sourceLine.RevertHitCount += revertHitCount
				sourceLine.IsCovered = sourceLine.IsCovered || sourceLine.SuccessHitCount > 0
				sourceLine.IsCoveredReverted = sourceLine.IsCoveredReverted || sourceLine.RevertHitCount > 0

			}
		} else {
			return fmt.Errorf("could not perform source code analysis, missing source '%v'", sourcePath)
		}

	}
	return nil
}

func determineLinesCovered(cm *ContractCoverageMap, bytecode []byte) ([]uint, []uint) {
	indexToOffset := getInstructionIndexToOffsetLookup(bytecode)
	jumpIndices := getJumpIndices(bytecode, indexToOffset)
	jumpDestIndices := getJumpDestIndices(bytecode, indexToOffset) // btw 0 should always go here, and the ones after jmps
	jumpToMarkers := getJumpToMarkers(indexToOffset, jumpIndices, jumpDestIndices)
	jumpDestToMarkers := getJumpDestToMarkers(indexToOffset, jumpIndices, jumpDestIndices)
	pcToRevertMarker := getRevertMarkers(indexToOffset)
	pcToReturnMarker := getReturnMarkers(indexToOffset)
	// interestingIndices := getInterestingIndices(indexToOffset, jumpIndices, jumpDestIndices, pcToRevertMarker, pcToReturnMarker, cm.successfulCoverage.executedFlags)

	fmt.Println(jumpIndices)
	fmt.Println(jumpDestIndices)

	execFlags := cm.successfulCoverage.executedFlags
	fmt.Println("-------------------")
	for k, v := range execFlags {
		fmt.Println(bits.RotateLeft64(k, 32) & 0xFFFFFFFF, k & 0xFFFFFFFF, v)
	}
	fmt.Println("===================")

	// panic("")

	successfulHits := make([]uint, len(indexToOffset))
	revertedHits := make([]uint, len(indexToOffset))

	hit := uint(0)
	fmt.Println("begin")
	for idx, pc := range indexToOffset {
		// if !interestingIndices[idx] { continue }
		if jumpDestIndices[idx] || jumpIndices[idx] {
			if jumpDestIndices[idx] { fmt.Println("jumpdest", hit, idx, bytecode[pc]) }
			if jumpIndices[idx] { fmt.Println("jump", hit, idx, bytecode[pc]) }
			hit = uint(0)
		} else {
			subtractOff := execFlags[pcToRevertMarker[idx]]
			if subtractOff > hit { // TODO if it's > that means we coded smth wrong
				fmt.Println("BAD CASE")
			}
			if subtractOff > 0 { fmt.Println("positive subtractoff", subtractOff, hit, idx) }
			hit -= subtractOff
		}

		if jumpDestIndices[idx] {
			for _, marker := range jumpDestToMarkers[idx] {
				hit += execFlags[marker]
				if execFlags[marker] > 0 { fmt.Println("positive jumpdest marker", execFlags[marker], hit, idx) }
			}
			/*for traverseIdx := idx+1; traverseIdx < len(indexToOffset) && !jumpIndices[traverseIdx] && !jumpDestIndices[traverseIdx]; traverseIdx++ {
				subtractOff := execFlags[pcToRevertMarker[traverseIdx]] + execFlags[pcToReturnMarker[traverseIdx]]
				if subtractOff >= hit { // TODO if it's > that means we coded smth wrong
					break
				}
				hit -= subtractOff
				successfulHits[traverseIdx] = hit
			}*/
		}
		if jumpIndices[idx] {
			for _, marker := range jumpToMarkers[idx] {
				hit += execFlags[marker]
				if execFlags[marker] > 0 { fmt.Println("positive jump marker", execFlags[marker], hit, idx) }
			}
		}

		hit += execFlags[uint64(pc)] // special case for when we just start out, TODO make this nicer
		if execFlags[uint64(pc)] > 0 { fmt.Println("positive zero", execFlags[uint64(pc)], hit, idx) }

		successfulHits[idx] = hit
		revertedHits[idx] = execFlags[pcToRevertMarker[idx]]
	
		subtractOff2 := execFlags[pcToReturnMarker[idx]]
		if subtractOff2 > 0 { fmt.Println("positive subtractoff2", subtractOff2, hit, idx) }
		if subtractOff2 > hit { // TODO if it's > that means we coded smth wrong
			fmt.Println("BAD CASE")
		}
		hit -= subtractOff2
	}

	return successfulHits, revertedHits
}

// GetInstructionIndexToOffsetLookup obtains a slice where each index of the slice corresponds to an instruction index,
// and the element of the slice represents the instruction offset.
// Returns the slice lookup, or an error if one occurs.
func getInstructionIndexToOffsetLookup(bytecode []byte) []int {
	// Create our resulting lookup
	indexToOffsetLookup := make([]int, 0, len(bytecode)/2)

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

func getJumpIndices(bytecode []byte, indexToOffset []int) map[int]bool {
        jumps := map[int]bool{}
	jumps[0] = true
        for idx, pc := range indexToOffset {
                op := vm.OpCode(bytecode[pc])
                if op == vm.JUMP || op == vm.JUMPI {
                        jumps[idx] = true
		}
        }
	return jumps
}

func getJumpDestIndices(bytecode []byte, indexToOffset []int) map[int]bool {
        jumpDests := map[int]bool{}
	if len(indexToOffset) > 0 {
		jumpDests[0] = true
	}
        for idx, pc := range indexToOffset {
                op := vm.OpCode(bytecode[pc])
                if op == vm.JUMPDEST {
                        jumpDests[idx] = true
                } else if op == vm.JUMPI && idx < len(indexToOffset) {
                        jumpDests[idx+1] = true
		}
        }
	return jumpDests
}

func getJumpToMarkers(indexToOffset []int, jumpIndices map[int]bool, jumpDestIndices map[int]bool) map[int][]uint64 {
	markers := map[int][]uint64{}
	// fmt.Println(jumpDestIndices)
	for jmp, _ := range jumpIndices {
		markersHere := make([]uint64, 0, len(jumpDestIndices))
		for dst, _ := range jumpDestIndices {
			// fmt.Println(indexToOffset, jmp, dst)
			markersHere = append(markersHere, bits.RotateLeft64(uint64(indexToOffset[jmp]), 32) ^ uint64(indexToOffset[dst]))
		}
		markers[jmp] = markersHere
	}
	return markers
}

func getJumpDestToMarkers(indexToOffset []int, jumpIndices map[int]bool, jumpDestIndices map[int]bool) map[int][]uint64 {
	markers := map[int][]uint64{}
	for dst, _ := range jumpDestIndices {
		markersHere := make([]uint64, 0, len(jumpIndices))
		for jmp, _ := range jumpIndices {
			markersHere = append(markersHere, bits.RotateLeft64(uint64(indexToOffset[jmp]), 32) ^ uint64(indexToOffset[dst]))
		}
		markers[dst] = markersHere
	}
	return markers
}

func getRevertMarkers(indexToOffset []int) []uint64 {
	markers := make([]uint64, len(indexToOffset))
	for idx, _ := range indexToOffset {
		markers[idx] = bits.RotateLeft64(uint64(idx), 32) ^ 0x40000000
	}
	return markers
}

func getReturnMarkers(indexToOffset []int) []uint64 {
	markers := make([]uint64, len(indexToOffset))
	for idx, _ := range indexToOffset {
		markers[idx] = bits.RotateLeft64(uint64(idx), 32) ^ 0x80000000
	}
	return markers
}

func getInterestingIndices(indexToOffset []int, jumpIndices map[int]bool, jumpDestIndices map[int]bool, pcToRevertMarker []uint64, pcToReturnMarker []uint64, executedFlags map[uint64]uint) map[int]bool {
	indices := map[int]bool{}
	for idx, _ := range indexToOffset {
		if jumpIndices[idx] || jumpDestIndices[idx] || executedFlags[pcToRevertMarker[idx]] > 0 || executedFlags[pcToReturnMarker[idx]] > 0 {
			indices[idx] = true
		}
	}
	return indices
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
