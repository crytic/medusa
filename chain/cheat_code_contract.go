package chain

import (
	"encoding/binary"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// cheatCodeContract defines a struct which represents a pre-compiled contract with various methods that is
// meant to act as a contract.
type cheatCodeContract struct {
	// address defines the address the cheat code contract should be installed at.
	address common.Address

	// tracer represents the cheat code tracer used to provide execution hooks.
	tracer *cheatCodeTracer

	// methodInfo describes a table of methodId (function selectors) to cheat code methods. This acts as a switch table
	// for different methods in the contract.
	methodInfo map[uint32]*cheatCodeMethod
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

// newCheatCodeContract returns a new precompiledContract which uses the attached cheatCodeTracer for execution
// context.
func newCheatCodeContract(tracer *cheatCodeTracer, address common.Address) *cheatCodeContract {
	return &cheatCodeContract{
		address:    address,
		tracer:     tracer,
		methodInfo: make(map[uint32]*cheatCodeMethod),
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

// addMethod adds a new method to the precompiled contract.
// Returns an error if one occurred.
func (p *cheatCodeContract) addMethod(name string, inputs abi.Arguments, outputs abi.Arguments, handler cheatCodeMethodHandler) {
	// Verify a method name was provided
	if name == "" {
		panic("could not add method to precompiled cheatcode contract, empty method name provided")
	}

	// Verify a method handler was provided
	if handler == nil {
		panic("could not add method to precompiled cheatcode contract, nil method handler provided")
	}

	// Set the method information in our method lookup
	method := abi.NewMethod(name, name, abi.Function, "external", false, false, inputs, outputs)
	methodId := binary.LittleEndian.Uint32(method.ID)
	p.methodInfo[methodId] = &cheatCodeMethod{
		method:  method,
		handler: handler,
	}
}

// RequiredGas determines the amount of gas necessary to execute the pre-compile with the given input data.
// Returns the gas cost.
func (p *cheatCodeContract) RequiredGas(input []byte) uint64 {
	return 0
}

// Run executes the given pre-compile with the provided input data.
// Returns the output data from execution, or an error if one occurred.
func (p *cheatCodeContract) Run(input []byte) ([]byte, error) {
	// Calling any method should require at least a signature
	if len(input) < 4 {
		return []byte{}, vm.ErrExecutionReverted
	}

	// Obtain the method identifier as a uint32
	methodId := binary.LittleEndian.Uint32(input[:4])

	// Ensure we have a method definition that matches our selector.
	methodInfo, methodInfoExists := p.methodInfo[methodId]
	if !methodInfoExists || methodId != binary.LittleEndian.Uint32(methodInfo.method.ID) {
		return []byte{}, vm.ErrExecutionReverted
	}

	// This call is targeting a valid method, unpack its arguments
	inputValues, err := methodInfo.method.Inputs.Unpack(input[4:])
	if err != nil {
		return []byte{}, vm.ErrExecutionReverted
	}

	// Call the registered method handler.
	outputValues, rawReturnData := methodInfo.handler(p.tracer, inputValues)

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
