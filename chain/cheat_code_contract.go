package chain

import (
	"encoding/binary"
	"fmt"

	"github.com/crytic/medusa/logging"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// CheatCodeContract defines a struct which represents a pre-compiled contract with various methods that is
// meant to act as a contract.
type CheatCodeContract struct {
	// The name of the cheat code contract.
	name string

	// address defines the address the cheat code contract should be installed at.
	address common.Address

	// tracer represents the cheat code tracer used to provide execution hooks.
	tracer *cheatCodeTracer

	// methodInfo describes a table of methodId (function selectors) to cheat code methods. This acts as a switch table
	// for different methods in the contract.
	methodInfo map[uint32]*cheatCodeMethod

	// abi refers to the cheat code contract's ABI definition.
	abi abi.ABI

	// storage holds values stored by cheatcodes
	storage map[string]map[string]any
}

// cheatCodeMethod defines the method information for a given precompiledContract.
type cheatCodeMethod struct {
	// method is the ABI method definition used to pack and unpack both input and output arguments.
	method abi.Method

	// handler represents the method handler to call with the unpacked input arguments
	handler cheatCodeMethodHandler
}

// cheatCodeMethodHandler describes a function which handles callback for a given contract method. It takes the
// cheatCodeTracer for execution context, as well as unpacked input values.
// Returns unpacked output values to be packed and returned, or raw return data values to use instead.
// Raw return data takes precedence over unpacked values, so only one should be returned.
type cheatCodeMethodHandler func(tracer *cheatCodeTracer, args []any) ([]any, *cheatCodeRawReturnData)

// cheatCodeRawReturnData represents the return data and/or error which should be returned by a cheatCodeMethodHandler.
type cheatCodeRawReturnData struct {
	// ReturnData represents the raw return data bytes to be sent to the caller. This is typically ABI encoded for
	// solidity, but it may be any data in practice.
	ReturnData []byte

	// Err represents an optional error which is to be returned to the caller. This may be vm.
	Err error
}

// getCheatCodeProviders obtains a cheatCodeTracer (used to power cheat code analysis) and associated CheatCodeContract
// objects linked to the tracer (providing on-chain callable methods as an entry point). These objects are attached to
// the TestChain to enable cheat code functionality.
// Returns the tracer and associated pre-compile contracts, or an error, if one occurred.
func getCheatCodeProviders() (*cheatCodeTracer, []*CheatCodeContract, error) {
	// Create a cheat code tracer and attach it to the chain.
	tracer := newCheatCodeTracer()

	// Obtain our standard cheat code pre-compile
	stdCheatCodeContract, err := getStandardCheatCodeContract(tracer)
	if err != nil {
		return nil, nil, err
	}

	// Obtain the console.log pre-compile
	consoleCheatCodeContract, err := getConsoleLogCheatCodeContract(tracer)
	if err != nil {
		return nil, nil, err
	}

	// Return the tracer and precompiles
	return tracer, []*CheatCodeContract{stdCheatCodeContract, consoleCheatCodeContract}, nil
}

// newCheatCodeContract returns a new precompiledContract which uses the attached cheatCodeTracer for execution
// context.
func newCheatCodeContract(tracer *cheatCodeTracer, address common.Address, name string) *CheatCodeContract {
	return &CheatCodeContract{
		name:       name,
		address:    address,
		tracer:     tracer,
		methodInfo: make(map[uint32]*cheatCodeMethod),
		abi: abi.ABI{
			Constructor: abi.Method{},
			Methods:     make(map[string]abi.Method),
			Events:      make(map[string]abi.Event),
			Errors:      make(map[string]abi.Error),
			Fallback:    abi.Method{},
			Receive:     abi.Method{},
		},
		storage: make(map[string]map[string]any),
	}
}

// cheatCodeRevertData creates cheatCodeRawReturnData with the provided return data, and an error indicating the
// EVM has encountered an execution revert (vm.ErrExecutionReverted).
func cheatCodeRevertData(returnData []byte) *cheatCodeRawReturnData {
	return &cheatCodeRawReturnData{
		ReturnData: returnData,
		Err:        vm.ErrExecutionReverted,
	}
}

// Name represents the name of the cheat code contract.
func (c *CheatCodeContract) Name() string {
	return c.name
}

// Address represents the address the cheat code contract is deployed at.
func (c *CheatCodeContract) Address() common.Address {
	return c.address
}

// Abi provides the cheat code contract interface.
func (c *CheatCodeContract) Abi() *abi.ABI {
	return &c.abi
}

// addMethod adds a new method to the precompiled contract.
// Throws a panic if either the name is the empty string or the handler is nil.
func (c *CheatCodeContract) addMethod(name string, inputs abi.Arguments, outputs abi.Arguments, handler cheatCodeMethodHandler) {
	// Verify a method name was provided
	if name == "" {
		logging.GlobalLogger.Panic("Failed to add method to precompile cheatcode contract", fmt.Errorf("empty method name provided"))
	}

	// Verify a method handler was provided
	if handler == nil {
		logging.GlobalLogger.Panic("Failed to add method to precompile cheatcode contract", fmt.Errorf("nil method handler provided"))
	}

	// Set the method information in our method lookup
	method := abi.NewMethod(name, name, abi.Function, "external", false, false, inputs, outputs)
	methodId := binary.LittleEndian.Uint32(method.ID)
	c.methodInfo[methodId] = &cheatCodeMethod{
		method:  method,
		handler: handler,
	}
	// Add the method to the ABI.
	// Note: Normally the key here should be the method name, not sig. But cheat code contracts have duplicate
	// method names with different parameter types, so we use this so they don't override.
	c.abi.Methods[method.Sig] = method
}

// RequiredGas determines the amount of gas necessary to execute the pre-compile with the given input data.
// Returns the gas cost.
func (c *CheatCodeContract) RequiredGas(input []byte) uint64 {
	return 0
}

// Run executes the given pre-compile with the provided input data.
// Returns the output data from execution, or an error if one occurred.
func (c *CheatCodeContract) Run(input []byte) ([]byte, error) {
	// Calling any method should require at least a signature
	if len(input) < 4 {
		return []byte{}, vm.ErrExecutionReverted
	}

	// Obtain the method identifier as an uint32
	methodId := binary.LittleEndian.Uint32(input[:4])

	// Ensure we have a method definition that matches our selector.
	methodInfo, methodInfoExists := c.methodInfo[methodId]
	if !methodInfoExists || methodId != binary.LittleEndian.Uint32(methodInfo.method.ID) {
		return []byte{}, vm.ErrExecutionReverted
	}

	// This call is targeting a valid method, unpack its arguments
	inputValues, err := methodInfo.method.Inputs.Unpack(input[4:])
	if err != nil {
		return []byte{}, vm.ErrExecutionReverted
	}

	// Call the registered method handler.
	outputValues, rawReturnData := methodInfo.handler(c.tracer, inputValues)

	// If we have raw return data, use that. Otherwise, proceed to unpack the returned output values.
	if rawReturnData != nil {
		return rawReturnData.ReturnData, rawReturnData.Err
	}

	// Pack our return values
	packedOutput, err := methodInfo.method.Outputs.Pack(outputValues...)
	if err != nil {
		return []byte{}, vm.ErrExecutionReverted
	}

	// Return our packed values
	return packedOutput, nil
}
