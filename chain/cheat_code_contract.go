package chain

import (
	"encoding/binary"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// cheatCodeMethodHandler describes a function which handles callback for a given contract method. It takes the
// cheatCodeTracer for execution context, as well as unpacked input values.
// Returns unpacked output values, or an error if one occurs.
type cheatCodeMethodHandler func(tracer *cheatCodeTracer, args []any) ([]any, error)

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

// newCheatCodeContract returns a new precompiledContract which uses the attached cheatCodeTracer for execution
// context.
func newCheatCodeContract(tracer *cheatCodeTracer, address common.Address) *cheatCodeContract {
	return &cheatCodeContract{
		address:    address,
		tracer:     tracer,
		methodInfo: make(map[uint32]*cheatCodeMethod),
	}
}

// addMethod adds a new method to the precompiled contract.
// Returns an errorw if one occurred.
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
		return nil, err
	}

	// Call the registered method handler.
	outputValues, err := methodInfo.handler(p.tracer, inputValues)
	if err != nil {
		return nil, err
	}

	// Return our packed output data.
	return methodInfo.method.Outputs.Pack(outputValues...)
}
