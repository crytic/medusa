package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/vm"
	"golang.org/x/exp/slices"
	"strings"
)

// CompiledContract represents a single contract unit from a smart contract compilation.
type CompiledContract struct {
	// Abi describes a contract's application binary interface, a structure used to describe information needed
	// to interact with the contract such as constructor and function definitions with input/output variable
	// information, event declarations, and fallback and receive methods.
	Abi abi.ABI

	// InitBytecode describes the bytecode used to deploy a contract.
	InitBytecode []byte

	// RuntimeBytecode represents the rudimentary bytecode to be expected once the contract has been successfully
	// deployed. This may differ at runtime based on constructor arguments, immutables, linked libraries, etc.
	RuntimeBytecode []byte

	// ParsedRuntimeBytecode represents an array of EVM instructions in the metadata-less bytecode
	// each instruction represented as the indices in the bytecode the EVM instruction spawns
	ParsedRuntimeBytecode []ParsedBytecodeData

	// SrcMapsInit describes the source mappings to associate source file and bytecode segments in InitBytecode.
	SrcMapsInit string

	// SrcMapsRuntime describes the source mappings to associate source file and bytecode segments in RuntimeBytecode.
	SrcMapsRuntime string

	// ParsedRuntimeSrcMap describes an array of source map elements represented as structs, each struct contains
	// all the fields expected in a source map element (i.e offset, length, fileID, etc...)
	ParsedRuntimeSrcMap []SrcMapElement

	// The sourcecode of the file represented as an array of bytes
	SourceCode []byte

	// SourceLines is an array of the "lines of source code" represented as two indices which indicate
	// where in the bytecode these lines begin and end (i.e the first line of some source file goes from index 0 to index 100)
	// it also includes an array of booleans indicating whether the byte at that index in the bytecode was covered or not
	SourceLines []SourceLine
}

// IsMatch returns a boolean indicating whether provided contract bytecode is a match to this compiled contract
// definition.
func (c *CompiledContract) IsMatch(initBytecode []byte, runtimeBytecode []byte) bool {
	// Check if we can compare init and runtime bytecode
	canCompareInit := len(initBytecode) > 0 && len(c.InitBytecode) > 0
	canCompareRuntime := len(runtimeBytecode) > 0 && len(c.RuntimeBytecode) > 0

	// First try matching runtime bytecode contract metadata.
	if canCompareRuntime {
		// First we try to match contracts with contract metadata embedded within the smart contract.
		// Note: We use runtime bytecode for this because init byte code can have matching metadata hashes for different
		// contracts.
		deploymentMetadata := ExtractContractMetadata(runtimeBytecode)
		definitionMetadata := ExtractContractMetadata(c.RuntimeBytecode)
		if deploymentMetadata != nil && definitionMetadata != nil {
			deploymentBytecodeHash := deploymentMetadata.ExtractBytecodeHash()
			definitionBytecodeHash := definitionMetadata.ExtractBytecodeHash()
			if deploymentBytecodeHash != nil && definitionBytecodeHash != nil {
				return bytes.Equal(deploymentBytecodeHash, definitionBytecodeHash)
			}
		}
	}

	// Since we could not match with runtime bytecode's metadata hashes, we try to match based on init code. To do this,
	// we anticipate our init bytecode might contain appended arguments, so we'll be slicing it down to size and trying
	// to match as a last ditch effort.
	if canCompareInit {
		// If the init byte code size is larger than what we initialized with, it is not a match.
		if len(c.InitBytecode) > len(initBytecode) {
			return false
		}

		// Cut down the contract init bytecode to the size of the definition's to attempt to strip away constructor
		// arguments before performing a direct compare.
		cutDeployedInitBytecode := initBytecode[:len(c.InitBytecode)]

		// If the byte code matches exactly, we treat this as a match.
		if bytes.Equal(cutDeployedInitBytecode, c.InitBytecode) {
			return true
		}
	}

	// As a final fallback, try to compare the whole runtime byte code (least likely to work, given the deployment
	// process, e.g. smart contract constructor, will change the runtime code in most cases).
	if canCompareRuntime {
		// If the byte code matches exactly, we treat this as a match.
		if bytes.Equal(runtimeBytecode, c.RuntimeBytecode) {
			return true
		}
	}

	// Otherwise return our failed match status.
	return false
}

// ParseABIFromInterface parses a generic object into an abi.ABI and returns it, or an error if one occurs.
func ParseABIFromInterface(i any) (*abi.ABI, error) {
	var (
		result abi.ABI
		err    error
	)

	// If it's a string, just parse it. Otherwise, we assume it's an interface and serialize it into a string.
	if s, ok := i.(string); ok {
		result, err = abi.JSON(strings.NewReader(s))
		if err != nil {
			return nil, err
		}
	} else {
		var b []byte
		b, err = json.Marshal(i)
		if err != nil {
			return nil, err
		}
		result, err = abi.JSON(strings.NewReader(string(b)))
		if err != nil {
			return nil, err
		}
	}
	return &result, nil
}

// GetDeploymentMessageData is a helper method used create contract deployment message data for the given contract.
// This data can be set in transaction/message structs "data" field to indicate the packed init bytecode and constructor
// argument data to use.
func (c *CompiledContract) GetDeploymentMessageData(args []any) ([]byte, error) {
	// ABI encode constructor arguments and append them to the end of the bytecode
	initBytecodeWithArgs := slices.Clone(c.InitBytecode)
	if len(c.Abi.Constructor.Inputs) > 0 {
		data, err := c.Abi.Pack("", args...)
		if err != nil {
			return nil, fmt.Errorf("could not encode constructor arguments due to error: %v", err)
		}
		initBytecodeWithArgs = append(initBytecodeWithArgs, data...)
	}
	return initBytecodeWithArgs, nil
}

// struct representing EVM operations in some bytecode, each operation is represented as the name of the instruction
// (i.e 'PUSH32'), the start and end indices it spawns in the bytecode and a boolean indicating whether it was covered or not
// NOTE The string and bool can probably be dropped after a bit of a clean-up of the code
type ParsedBytecodeData struct {
	Instruction string
	Start       int
	End         int
	Covered     bool
}

// Given some bytecode we split it into the different EVM operations performed in said bytecode
// the returned data is an array of structs containing the EVM instruction as a string
// and the start and end offsets the EVM instruction spawns in the bytecode
// NOTE pass a metada-less bytecode
func ParseBytecode(bytecode []byte) []ParsedBytecodeData {
	var parsedBytecode []ParsedBytecodeData

	// traverse the bytecode
	i := 0

	for i < len(bytecode) {
		// get instruction at index
		instruction := bytecode[i]

		// if instruction is a PUSH opcode
		if vm.OpCode(instruction).IsPush() {
			// figure out which kind of PUSH instruction it is (i.e PUSH1 - PUSH32)
			// NOTE they might introduce PUSH0 so we might want to revisit this function when that happens
			amountOfBytesPushed := int(instruction) - 95

			// store the range of bytes that the instruction spawns
			// e.g if the first instruction is PUSH32 then the range it spawns is [0, 31]
			instructionString := vm.OpCode(instruction).String()
			parsedBytecode = append(parsedBytecode, ParsedBytecodeData{instructionString, i, i + amountOfBytesPushed, false})

			i = i + amountOfBytesPushed + 1
		} else {
			// if it is not a PUSH then it only spawns a single byte
			instructionString := vm.OpCode(instruction).String()
			parsedBytecode = append(parsedBytecode, ParsedBytecodeData{instructionString, i, i, false})

			i = i + 1
		}
	}
	return parsedBytecode
}

// Represents a source map element and whether it was covered or not
// NOTE we mark source map elements as covered under the assumption that they cannot be partially covered
// If it was the case that they can be partially covered then a more nuanced coverage is needed
type SrcMapElement struct {
	Offset   string
	Length   string
	FileID   string
	JumpType string
	ModPath  string
	Covered  bool
}

// Given some source map represented as a string it returns an array of source map elements (as described above)
// The parsing is performed according to the rules specified in the solidity documentation
// https://docs.soliditylang.org/en/latest/internals/source_mappings.html which state:

// In order to compress these source mappings especially for bytecode, the following rules are used:
// 1. If a field is empty, the value of the preceding element is used.
// 1. If a ':' is missing, all following fields are considered empty.

// Following these rules we first separate the source map into all its invidivual elements (i.e each ';' delimits an element)
// an element looks like "s:l:f:j:m", using rule 1 as the baseline we check wether the element is empty
// if its empty we can assume the element is the same as the previous element

// After applying the first rule we split each element into its individual fiels (i.e "s", "l, "f", "j", "m")
// Once we do that we can apply the second rule which says that if a ':' is missing then all the following fiels are empty
// when a field is empty it has the same values as the one before it

// NOTE in this particular case we add repetitive elements based on these rules; however we could add a count to the elements
// that way we don't have to occupy that space
func ParseSourceMap(sourceMap string) []SrcMapElement {
	var parsedSourceMap []SrcMapElement

	if len(sourceMap) > 0 {
		// Separate into all the invidivual source mapping elements
		elements := strings.Split(sourceMap, ";")

		// We take the first element of the source map as the baseline and split it into its subelements
		// we can do this because the first element can never be empty
		ep := strings.Split(elements[0], ":")

		// We use this variable to store "the previous element" because the way
		// the source mapping works when an element or field is "empty"
		// the value of the previous element is used
		previous := SrcMapElement{ep[0], ep[1], ep[2], ep[3], ep[4], false}

		// iterate over all elements
		for _, element := range elements {
			// if the element is empty it means its the same as the previous one
			if element == "" {
				parsedSourceMap = append(parsedSourceMap, previous)
				// the element is not empty however it can still have some of its fields empty
			} else {
				fields := strings.Split(element, ":")

				// TODO improve this little "algorithm"
				for i, value := range fields {
					if i == 0 && value != "" {
						previous.Offset = value
					} else if i == 1 && value != "" {
						previous.Length = value
					} else if i == 2 && value != "" {
						previous.FileID = value
					} else if i == 3 && value != "" {
						previous.JumpType = value
					} else if i == 4 && value != "" {
						previous.ModPath = value
					}
				}

				parsedSourceMap = append(parsedSourceMap, previous)
			}
		}
	}
	return parsedSourceMap
}

// NOTE from this point on these functions might be better suited to live somewhere else in the codebase (i.e in a different package)

// represents a line of source code as the index at which it begins and the index at which it ends
// it also contained an array of boolean indicating whether the byte at that index was covered or not
// NOTE a bol is enough for "covered || not covered" if we want more nuanced coverage we might want to use
// a different type
type SourceLine struct {
	Begin   int
	End     int
	Covered []bool
}

// Iterate over the source file data and figures out where each line of the source file starts and ends
func SplitSourceFileIntoLines(fileData []byte) []SourceLine {
	var enrichedData []SourceLine
	var lineStart int

	for i, data := range fileData {
		if data == byte('\n') {
			enrichedData = append(enrichedData, SourceLine{lineStart, i, make([]bool, i+1-lineStart)})

			lineStart = i + 1
		}
	}

	return enrichedData
}
