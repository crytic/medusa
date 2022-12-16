package slither

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// SlitherData is the data structure that holds the results from slither's echidna printer
// TODO: Remove all parameters that will not be used
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
	// TODO: Don't think we need this
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
	// SolcVersions holds all the solc versions used during compilation (TODO: can remove this one?)
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
type Constant struct {
	// Contract represents the name of the contract from where the constant was derived
	Contract string
	// Method represents the name of the method from where the constant was derived
	Method string
	// Type represents what kind of constant this is (e.g. uint256, string, bytes32, etc.)
	Type *abi.Type
	// Value is the string representation of the constant
	// TODO: This should maybe become `any` instead of `string`
	Value string
}

// newConstant will create a new Constant
func newConstant(contract string, method string, constantType *abi.Type, constantValue string) *Constant {
	return &Constant{
		Contract: contract,
		Method:   method,
		Type:     constantType,
		Value:    constantValue,
	}
}

// makeConstantsList creates a list of Constant given a rawConstants object. This is used to set the SlitherData.Constants
// parameter which will be used to query for constants
func makeConstantsList(constantsUsed rawConstants) ([]*Constant, error) {
	// Create a list of Constants
	constants := make([]*Constant, 0)

	// Iterate through each contract in the rawConstants object
	for contract, constantsInContract := range constantsUsed {
		// Iterate through each method in the contract
		for method, constantsInMethod := range constantsInContract {
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
					// Note: There is an assumption here that slither is not returning anything but an elementary type
					abiType, err := abi.NewType(slitherType, "", nil)
					if err != nil {
						return nil, err
					}

					// Create a new constant and add it to the list
					constant := newConstant(contract, method, &abiType, value)
					constants = append(constants, constant)
				}
			}
		}
	}

	return constants, nil
}

// GetConstantsByType will return all the constants for a specific abi.Type.
// Note that the `T` field in abi.Type will be used for type equality checking
func (s *SlitherData) GetConstantsByType(abiType *abi.Type) []*Constant {
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
func (s *SlitherData) GetConstantsInContract(contract string) []*Constant {
	constantsInContract := make([]*Constant, 0)

	// For each constant, check whether the contract associated with the constant is the same as the provided input
	for _, constant := range s.Constants {
		if constant.Contract == contract {
			constantsInContract = append(constantsInContract, constant)
		}
	}

	return constantsInContract
}

// GetConstantsInContractByType will return all the constants associated with a given contract for a specific type
// Note that the `T` field in abi.Type will be used for type equality checking
func (s *SlitherData) GetConstantsInContractByType(contract string, abiType *abi.Type) []*Constant {
	constantsInContractWithAbiType := make([]*Constant, 0)

	// For each constant, check whether the contract and type associated with the constant is the same as the provided inputs
	for _, constant := range s.Constants {
		if constant.Contract == contract && constant.Type.T == abiType.T {
			constantsInContractWithAbiType = append(constantsInContractWithAbiType, constant)
		}
	}

	return constantsInContractWithAbiType
}

// GetConstantsInMethod will return all the constants associated with a given (contract, method) tuple
func (s *SlitherData) GetConstantsInMethod(contract string, method string) []*Constant {
	constantsInMethod := make([]*Constant, 0)

	// For each constant, check whether the contract and method associated with the constant is the same as the provided inputs
	for _, constant := range s.Constants {
		if constant.Contract == contract && constant.Method == method {
			constantsInMethod = append(constantsInMethod, constant)
		}
	}

	return constantsInMethod
}

// GetConstantsInMethod will return all the constants associated with a given (contract, method) tuple for a specific type.
// Note that the `T` field in abi.Type will be used for type equality checking.
func (s *SlitherData) GetConstantsInMethodByType(contract string, method string, abiType *abi.Type) []*Constant {
	constantsInMethodWithAbiType := make([]*Constant, 0)

	// For each constant, check whether the contract, method, and type associated with the constant is the same as the provided inputs
	for _, constant := range s.Constants {
		if constant.Contract == contract && constant.Method == method && constant.Type.T == abiType.T {
			constantsInMethodWithAbiType = append(constantsInMethodWithAbiType, constant)
		}
	}

	return constantsInMethodWithAbiType
}
