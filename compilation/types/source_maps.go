package types

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/core/vm"
)

// Reference: Source mapping is performed according to the rules specified in solidity documentation:
// https://docs.soliditylang.org/en/latest/internals/source_mappings.html

// SourceMapJumpType describes the type of jump operation occurring within a SourceMapElement if the instruction
// is jumping.
type SourceMapJumpType string

const (
	// SourceMapJumpTypeNone indicates no jump occurred.
	SourceMapJumpTypeNone SourceMapJumpType = ""

	// SourceMapJumpTypeJumpIn indicates a jump into a function occurred.
	SourceMapJumpTypeJumpIn SourceMapJumpType = "i"

	// SourceMapJumpTypeJumpOut indicates a return from a function occurred.
	SourceMapJumpTypeJumpOut SourceMapJumpType = "o"

	// SourceMapJumpTypeJumpWithin indicates a jump occurred within the same function, e.g. for loops.
	SourceMapJumpTypeJumpWithin SourceMapJumpType = "-"
)

// SourceMap describes a list of elements which correspond to instruction indexes in compiled bytecode, describing
// which source files and the start/end range of the source code which the instruction maps to.
type SourceMap []SourceMapElement

// SourceMapElement describes an individual element of a source mapping output by the compiler.
// The index of each element in a source map corresponds to an instruction index (not to be mistaken with offset).
// It describes portion of a source file the instruction references.
type SourceMapElement struct {
	// Index refers to the index of the SourceMapElement within its parent SourceMap. This is not actually a field
	// saved in the SourceMap, but is provided for convenience so the user may remove SourceMapElement objects during
	// analysis.
	Index int

	// Offset refers to the byte offset which marks the start of the source range the instruction maps to.
	Offset int

	// Length refers to the byte length of the source range the instruction maps to.
	Length int

	// SourceUnitID refers to an identifier for the CompiledSource file which houses the relevant source code.
	SourceUnitID int

	// JumpType refers to the SourceMapJumpType which provides information about any type of jump that occurred.
	JumpType SourceMapJumpType

	// ModifierDepth refers to the depth in which code has executed a modifier function. This is used to assist
	// debuggers, e.g. understanding if the same modifier is re-used multiple times in a call.
	ModifierDepth int
}

// ParseSourceMap takes a source mapping string returned by the compiler and parses it into an array of
// SourceMapElement objects.
// Returns the list of SourceMapElement objects.
func ParseSourceMap(sourceMapStr string) (SourceMap, error) {
	// Define our variables to store our results in
	var (
		sourceMap SourceMap
		err       error
	)

	// If our provided source map string is empty, there is no work to be done.
	if len(sourceMapStr) == 0 {
		return sourceMap, nil
	}

	// Separate all the individual source mapping elements
	elements := strings.Split(sourceMapStr, ";")

	// We use this variable to store "the previous element" because the way
	// the source mapping works when an element or field is "empty"
	// the value of the previous element is used.
	current := SourceMapElement{
		Index:         -1,
		Offset:        -1,
		Length:        -1,
		SourceUnitID:  -1,
		JumpType:      "",
		ModifierDepth: 0,
	}

	// Iterate over all elements split from the source mapping
	for _, element := range elements {
		// Set the current index
		current.Index = len(sourceMap)

		// If the element is empty, we use the previous one
		if len(element) == 0 {
			sourceMap = append(sourceMap, current)
			continue
		}

		// Split the element fields apart
		fields := strings.Split(element, ":")

		// If the source range start offset exists, update our current element data.
		if len(fields) > 0 && fields[0] != "" {
			current.Offset, err = strconv.Atoi(fields[0])
			if err != nil {
				return nil, err
			}
		}

		// If the source range length exists, update our current element data.
		if len(fields) > 1 && fields[1] != "" {
			current.Length, err = strconv.Atoi(fields[1])
			if err != nil {
				return nil, err
			}
		}

		// If the source file identifier exists, update our current element data.
		if len(fields) > 2 && fields[2] != "" {
			current.SourceUnitID, err = strconv.Atoi(fields[2])
			if err != nil {
				return nil, err
			}
		}

		// If the jump type information exists, update our current element data.
		if len(fields) > 3 && fields[3] != "" {
			current.JumpType = SourceMapJumpType(fields[3])
		}

		// If the modifier call depth exists, update our current element data.
		if len(fields) > 4 && fields[4] != "" {
			current.ModifierDepth, err = strconv.Atoi(fields[4])
			if err != nil {
				return nil, err
			}
		}

		// Append our element to the map
		sourceMap = append(sourceMap, current)
	}

	// Return the resulting map
	return sourceMap, nil
}

// GetInstructionIndexToOffsetLookup obtains a slice where each index of the slice corresponds to an instruction index,
// and the element of the slice represents the instruction offset.
// Returns the slice lookup, or an error if one occurs.
func (s SourceMap) GetInstructionIndexToOffsetLookup(bytecode []byte) ([]int, error) {
	// Create our resulting lookup
	indexToOffsetLookup := make([]int, len(s))

	// Loop through all byte code
	currentOffset := 0
	for i := 0; i < len(indexToOffsetLookup); i++ {
		// If we're going to read out of bounds, return an error.
		if currentOffset >= len(bytecode) {
			return nil, fmt.Errorf("failed to obtain a lookup of instruction indexes to offsets. instruction index: %v, current offset: %v, length: %v", i, currentOffset, len(bytecode))
		}

		// Obtain the indexed instruction and add the current offset to our lookup at this index.
		op := vm.OpCode(bytecode[currentOffset])
		indexToOffsetLookup[i] = currentOffset

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
	return indexToOffsetLookup, nil
}
