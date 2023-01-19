package slither

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/trailofbits/medusa/fuzzing/valuegeneration"
	"github.com/trailofbits/medusa/utils"
	"math/big"
	"os/exec"
)

// SlitherData is the data structure that holds the results from slither's echidna printer
// TODO: Which of these parameters can we get rid off? Should we just keep everything?
type SlitherData struct {
	// Payable holds all the functions that are marked as payable
	Payable map[string][]string `json:"payable"`
	// Timestamp holds all the functions that use `block.timestamp`
	Timestamp map[string][]string `json:"timestamp"`
	// BlockNumber holds all the functions that use `block.number`
	BlockNumber map[string][]string `json:"block_number"`
	// MsgSender holds all the functions that use `msg.sender`
	MsgSender map[string][]string `json:"msg_sender"`
	// MsgGas holds all the functions that use `msg.gas`
	MsgGas map[string][]string `json:"msg_gas"`
	// Assert holds all the functions that have an `assert` statement
	Assert map[string][]string `json:"assert"`
	// ConstantFunctions holds all constant functions / variables
	ConstantFunctions map[string][]string `json:"constant_functions"`
	// ConstantsUsed holds all the various constants identified in all the functions
	ConstantsUsed rawConstants `json:"constants_used"`
	// ConstantsUsedInBinary holds all the various constants identified while evaluating unary and binary operations
	ConstantsUsedInBinary map[string]any `json:"constants_used_in_binary"`
	// Constants holds the list of Constant that should be used to query for constants given a specific type, contract, method, etc.
	Constants []*Constant
	// FunctionsRelations holds information about which functions impact another
	FunctionsRelations map[string]any `json:"functions_relations"`
	// Constructors holds which contracts have constructors
	Constructors map[string]string `json:"constructors"`
	// HaveExternalCalls holds which functions have external calls to another
	HaveExternalCalls map[string]any `json:"have_external_calls"`
	// CallAParameter has something to do with parameters
	CallAParameter map[string]any `json:"call_a_parameter"`
	// UseBalance holds all the function have an eth balance check
	UseBalance map[string]any `json:"use_balance"`
	// SolcVersions holds all the solc versions used during compilation
	SolcVersions []string `json:"solc_versions"`
	// WithFallback holds all the contracts with a fallback function
	WithFallback []string `json:"with_fallback"`
	// WithReceive holds all the contracts with a receive() ETH function
	WithReceive []string `json:"with_receive"`
}

// rawConstants is the "raw" format for the constants identified by slither. It is called "raw" because the
// format has an extra nesting of lists which makes it difficult to use. This map will be parsed into
// the SlitherData.Constants parameter which is what we recommend using to grab any constants.
// The format of this map is contractName -> methodName -> list of list of constants
type rawConstants map[string]map[string][][]map[string]string

// Constant represents a constant that was identified by slither. Each Constant can be mapped to the specific contract
// and method where the constant was found.
// TODO: Can we get rid of the contractName and methodName fields?
type Constant struct {
	// ContractName represents the name of the contract from where the constant was derived
	ContractName string
	// MethodName represents the name of the method from where the constant was derived
	MethodName string
	// Type represents what kind of constant this is (e.g. uint256, string, bytes32, etc.)
	Type abi.Type
	// Value is the string representation of the value.
	Value string
}

// newConstant will create a new Constant
func newConstant(contractName string, methodName string, constantType abi.Type, constantValue string) *Constant {
	return &Constant{
		ContractName: contractName,
		MethodName:   methodName,
		Type:         constantType,
		Value:        constantValue,
	}
}

// RunPrinter will run Slither's echidna printer and returns an object that will store the output data.
// Note that the function will not compile the target since it is expected that the compilation artifacts already exist
// at the target location.
func RunPrinter(target string) (*SlitherData, error) {
	// Make sure target is not the empty string
	if target == "" {
		return nil, fmt.Errorf("must provide a target to run slither's echidna printer")
	}

	// Set up the arguments necessary for a no-compile slither printer run
	args := []string{target, "--print", "echidna", "--ignore-compile", "--json", "-"}

	// Run the command
	cmd := exec.Command("slither", args...)
	cmdStdout, _, cmdCombined, err := utils.RunCommandWithOutputAndError(cmd)

	// If we failed, exit out
	if err != nil {
		return nil, fmt.Errorf("error while running slither:\n%s\n\nCommand Output:\n%s\n", err.Error(), string(cmdCombined))
	}

	// The actual printer data will be stored in the results[`results`][`printers`] key. The value of `printers` is
	// actually a _list_ of mappings where the `description` key at the 0th index will hold the data for the `echidna` printer
	type rawSlitherOutput struct {
		Error   any                         `json:"error"`
		Results map[string][]map[string]any `json:"results"`
	}

	// Unmarshal stdout
	var rawOutput rawSlitherOutput
	err = json.Unmarshal(cmdStdout, &rawOutput)
	if err != nil {
		return nil, fmt.Errorf("error while unmarshaling slither's output: %v\n", err.Error())
	}

	// If, for some reason, the printer or slither failed to run, exit out
	if rawOutput.Error != nil {
		return nil, fmt.Errorf("slither returned the following error: %v\n", rawOutput.Error)
	}
	rawResults := rawOutput.Results

	// Make sure there is only one printer result and those results are from the `echidna` printer
	if len(rawResults["printers"]) != 1 || rawResults["printers"][0]["printer"] != ECHIDNA_PRINTER {
		return nil, fmt.Errorf("expected the slither output to contain the results from the echidna printer")
	}

	var slitherData *SlitherData
	// Unmarshal the data into a SlitherData type
	err = json.Unmarshal([]byte(rawResults["printers"][0]["description"].(string)), &slitherData)
	if err != nil {
		return nil, fmt.Errorf("error while unmarshaling slither's echidna printer results: %v\n", err.Error())
	}

	// Create the parse-able version of all the constants found in the target
	slitherData.Constants, err = makeConstantsList(slitherData.ConstantsUsed)
	if err != nil {
		return nil, fmt.Errorf("error while parsing the constants in the output: %v\n", err.Error())
	}

	return slitherData, nil
}

// makeConstantsList creates a list of Constant given a rawConstants object. This is used to set the SlitherData.Constants
// parameter which will be used to query for constants
func makeConstantsList(constantsUsed rawConstants) ([]*Constant, error) {
	// Create a list of Constants
	constants := make([]*Constant, 0)

	// Iterate through each contract in the rawConstants object
	for contractName, constantsInContract := range constantsUsed {
		// Iterate through each method in the contract
		for methodName, constantsInMethod := range constantsInContract {
			// Iterate through each constant in a method
			for _, constantsList := range constantsInMethod {
				for _, rawConstant := range constantsList {
					// Grab the `type` and `value` of the constant
					slitherType, ok := rawConstant["type"]
					if !ok {
						return nil, fmt.Errorf("cannot find the `type` key in the following constant: %v\n", rawConstant)
					}
					value, ok := rawConstant["value"]
					if !ok {
						return nil, fmt.Errorf("cannot find the `value` key in the following constant: %v\n", rawConstant)
					}

					// Get the abi.Type for the given slitherType
					abiType, err := abi.NewType(slitherType, "", nil)
					if err != nil {
						return nil, err
					}

					// Create a new constant and add it to the list
					constant := newConstant(contractName, methodName, abiType, value)
					constants = append(constants, constant)
				}
			}
		}
	}

	return constants, nil
}

// AddConstantsToValueSet will add all the constants in SlitherData.Constants to the value set provided as an argument.
func (s *SlitherData) AddConstantsToValueSet(vs *valuegeneration.ValueSet) error {
	// Guard clause to make sure that the Constants field has been set
	if s.Constants == nil {
		return fmt.Errorf("there are no constants to add to the base value set\n")
	}

	// Iterate across all constants
	for _, constant := range s.Constants {
		switch constant.Type.T {
		case abi.IntTy, abi.UintTy:
			// Add the int / uint to value set
			b, success := new(big.Int).SetString(constant.Value, 10)
			if !success {
				return fmt.Errorf("unable to convert %v into a base-10 integer\n", constant.Value)
			}
			vs.AddInteger(b)
		case abi.AddressTy:
			// Add address to value set
			addr, err := utils.HexStringToAddress(constant.Value)
			if err != nil {
				return err
			}
			vs.AddAddress(addr)
		case abi.StringTy:
			// Add string to value set
			vs.AddString(constant.Value)
		case abi.BytesTy, abi.FixedBytesTy:
			// Add dynamic / fixed bytes to value set
			vs.AddBytes([]byte(constant.Value))
		default:
			return fmt.Errorf("invalid abi type identified for slither constant:%v\n", constant.Type.T)
		}
	}
	return nil
}

// TODO: Part of me feels like we can get rid of all the function below this TODO.

// GetConstantsByType will return all the constants for a specific abi.Type.
// Note that the `T` field in abi.Type will be used for type equality checking
func (s *SlitherData) GetConstantsByType(abiType abi.Type) []*Constant {
	constantsWithAbiType := make([]*Constant, 0)

	// For each constant, check whether the type associated with the constant is the same as the provided input
	for _, constant := range s.Constants {
		if constant.Type.T == abiType.T {
			constantsWithAbiType = append(constantsWithAbiType, constant)
		}
	}

	return constantsWithAbiType
}

// GetConstantsInContract will return all the constants associated with a given contract
func (s *SlitherData) GetConstantsInContract(contractName string) []*Constant {
	constantsInContract := make([]*Constant, 0)

	// For each constant, check whether the contract associated with the constant is the same as the provided input
	for _, constant := range s.Constants {
		if constant.ContractName == contractName {
			constantsInContract = append(constantsInContract, constant)
		}
	}

	return constantsInContract
}

// GetConstantsInContractByType will return all the constants associated with a given contract for a specific type
// Note that the `T` field in abi.Type will be used for type equality checking
func (s *SlitherData) GetConstantsInContractByType(contractName string, abiType abi.Type) []*Constant {
	constantsInContractWithAbiType := make([]*Constant, 0)

	// For each constant, check whether the contract and type associated with the constant is the same as the provided inputs
	for _, constant := range s.Constants {
		if constant.ContractName == contractName && constant.Type.T == abiType.T {
			constantsInContractWithAbiType = append(constantsInContractWithAbiType, constant)
		}
	}

	return constantsInContractWithAbiType
}

// GetConstantsInMethod will return all the constants associated with a given (contract, method) tuple
func (s *SlitherData) GetConstantsInMethod(contractName string, methodName string) []*Constant {
	constantsInMethod := make([]*Constant, 0)

	// For each constant, check whether the contract and method associated with the constant is the same as the provided inputs
	for _, constant := range s.Constants {
		if constant.ContractName == contractName && constant.MethodName == methodName {
			constantsInMethod = append(constantsInMethod, constant)
		}
	}

	return constantsInMethod
}

// GetConstantsInMethod will return all the constants associated with a given (contract, method) tuple for a specific type.
// Note that the `T` field in abi.Type will be used for type equality checking.
func (s *SlitherData) GetConstantsInMethodByType(contractName string, methodName string, abiType abi.Type) []*Constant {
	constantsInMethodWithAbiType := make([]*Constant, 0)

	// For each constant, check whether the contract, method, and type associated with the constant is the same as the provided inputs
	for _, constant := range s.Constants {
		if constant.ContractName == contractName && constant.MethodName == methodName && constant.Type.T == abiType.T {
			constantsInMethodWithAbiType = append(constantsInMethodWithAbiType, constant)
		}
	}

	return constantsInMethodWithAbiType
}
