package slither

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/trailofbits/medusa/fuzzing/valuegeneration"
	"github.com/trailofbits/medusa/utils"
	"math/big"
)

// SlitherData is the data structure that holds the results from slither's echidna printer
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
	ConstantsUsed rawConstantsSet `json:"constants_used"`
	// ConstantsUsedInBinary holds all the various constants identified in all the functions (TODO: fix this)
	ConstantsUsedInBinary map[string]any `json:"constants_used_in_binary"`
	// ConstantsSet holds all the constants in a query-able and iterable fashion. It is equivalent to the values held in ConstantsUsed
	AllConstants *ConstantsSet
	// FunctionsRelations holds information about which functions impact another
	FunctionsRelations map[string]any `json:"functions_relations"`
	// Constructors holds which contracts have constructors
	Constructors map[string]string `json:"constructors"`
	// HaveExternalCalls holds which functions have external calls to another
	HaveExternalCalls map[string]any `json:"have_external_calls"`
	// CallAParameter has something to do with parameters (TODO: fix this)
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

// rawConstantsSet holds is the "raw" format for the constants identified by slither. It is called "raw" because the
// format has an extra nesting of lists which makes it difficult to use. This map will be parsed into
// a ConstantsSet which is what we recommend using to grab any constants.
// The format of this map is contractName -> methodName -> list of list of constants
type rawConstantsSet map[string]map[string][][]map[string]string

// ConstantsSet holds all the constants that were identified by slither. The constant set is broken down by
// each contract and then method. Note that ConstantsSet does not guarantee uniqueness of constants. If the same type+value
// applies for two different methods in the same contract, that constant will not be de-duplicated.
type ConstantsSet struct {
	// Contracts is a mapping between contract name to a ContractConstants
	Contracts map[string]*ContractConstants
}

// newConstantsSet creates a new ConstantsSet given a rawConstantsSet. This will be used to set the SlitherData.AllConstants
// field.
func newConstantsSet(rawConstantsSet rawConstantsSet) (*ConstantsSet, error) {
	// Create a new ConstantsSet
	constantsSet := &ConstantsSet{
		Contracts: make(map[string]*ContractConstants, 0),
	}

	// Iterate through each contract
	for contractName, constantsInContract := range rawConstantsSet {
		// Create a new object to store the constants in a contract
		constantsSet.Contracts[contractName] = newContractConstants()
		contract := constantsSet.Contracts[contractName]
		// Iterate through each method in the contract
		for methodName, constantsInMethod := range constantsInContract {
			// Create a new object to store the constants in a method
			contract.Methods[methodName] = newMethodConstants()
			method := contract.Methods[methodName]
			// Iterate through each constant in a method
			for _, constantsList := range constantsInMethod {
				for _, constant := range constantsList {
					// Grab the `type` and `value` of the constant
					slitherType, ok := constant["type"]
					if !ok {
						return nil, fmt.Errorf("unable to grab the type of a constant in %v.%v\n", contractName, method)
					}
					value, ok := constant["value"]
					if !ok {
						return nil, fmt.Errorf("unable to grab the value of a constant in %v.%v\n", contractName, method)
					}
					// Get the abi.Type for the given slitherType, which is a string
					// TODO: There is an assumption here that slither is not returning anything but an elementary type
					//  (e.g., uintX, intX, byteX, address, etc.)
					abiType, err := abi.NewType(slitherType, "", nil)
					if err != nil {
						return nil, err
					}
					// Add the constant to the set
					method.Constants = append(method.Constants, newConstant(&abiType, value))
				}
			}
		}
	}

	return constantsSet, nil
}

// ContractConstants holds all the constants identified by slither in a specific contract
type ContractConstants struct {
	// Methods is a mapping between method name to a MethodConstants
	Methods map[string]*MethodConstants
}

// newContractConstants will create a new ContractConstants object
func newContractConstants() *ContractConstants {
	return &ContractConstants{
		Methods: make(map[string]*MethodConstants, 0),
	}
}

// MethodConstants holds all the constants identified by slither in a specific method
type MethodConstants struct {
	// Constants is a list of Constant
	Constants []*Constant
}

// newMethodConstants will create a new MethodConstants object
func newMethodConstants() *MethodConstants {
	return &MethodConstants{
		Constants: make([]*Constant, 0),
	}
}

// Constant is a specific constant identified by slither.
type Constant struct {
	// Type holds the type of the constant (e.g. uint256, string, bytes32, etc.)
	Type *abi.Type
	// Value holds the actual constant value
	Value any
}

// newConstant will create a new Constant
func newConstant(constantType *abi.Type, constantValue string) *Constant {
	return &Constant{
		Type:  constantType,
		Value: constantValue,
	}
}

// GetConstantsInContract will return all the constants associated with a given contract
func (s *SlitherData) GetConstantsInContract(contractName string) []*Constant {
	// Create an array to hold the constants
	constants := make([]*Constant, 0)

	// Append all the constants for all the methods in the given contractName
	for _, constantsInMethod := range s.AllConstants.Contracts[contractName].Methods {
		constants = append(constants, constantsInMethod.Constants...)
	}

	return constants
}

// GetConstantsInMethod will return all the constants associated with a given method
func (s *SlitherData) GetConstantsInMethod(contractName string, methodName string) []*Constant {
	// Just return all the constants in the given contractName:methodName combination
	return s.AllConstants.Contracts[contractName].Methods[methodName].Constants
}

// AddConstantsToValueSet will add all the constants in the ConstantsSet to a given valuegeneration.ValueSet.
func (s *SlitherData) AddConstantsToValueSet(vs *valuegeneration.ValueSet) error {
	// Iterate through each contract in the set
	for _, constantsInContract := range s.AllConstants.Contracts {
		// Iterate through each method in the contract
		for _, constantsInMethod := range constantsInContract.Methods {
			// Iterate through each constant identified in the method
			for _, constant := range constantsInMethod.Constants {
				// Add the constant to the value set
				err := addConstantToValueSet(vs, constant)
				if err != nil {
					return fmt.Errorf("error while adding a constant to the value set: %v\n", err.Error())
				}
			}
		}
	}
	return nil
}

// addConstantToValueSet is a helper function add a constant to a given valuegeneration.ValueSet.
func addConstantToValueSet(vs *valuegeneration.ValueSet, constant *Constant) error {
	// Add uintX or intX to the value set
	switch constant.Type.T {
	case abi.IntTy:
		b, success := new(big.Int).SetString(constant.Value.(string), 10)
		if !success {
			return fmt.Errorf("unable to convert %v into a base-10 integer\n", constant.Value)
		}
		vs.AddInteger(b)
	case abi.UintTy:
		b, success := new(big.Int).SetString(constant.Value.(string), 10)
		if !success {
			return fmt.Errorf("unable to convert %v into a base-10 integer\n", constant.Value)
		}
		vs.AddInteger(b)
	case abi.AddressTy:
		addr, err := utils.HexStringToAddress(constant.Value.(string))
		if err != nil {
			return err
		}
		vs.AddAddress(addr)
	case abi.StringTy:
		vs.AddString(constant.Value.(string))
	case abi.BytesTy:
		vs.AddBytes([]byte(constant.Value.(string)))
	case abi.FixedBytesTy:
		vs.AddBytes([]byte(constant.Value.(string)))
	default:
		return fmt.Errorf("invalid abi type identified for slither constant:%v\n", constant.Type.T)
	}

	return nil
}
