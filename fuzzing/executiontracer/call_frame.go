package executiontracer

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// CallFrames represents a list of call frames recorded by the ExecutionTracer.
type CallFrames []*CallFrame

// CallFrame contains information on each EVM call scope, as recorded by an ExecutionTracer.
type CallFrame struct {
	// SenderAddress refers to the address which produced this call.
	SenderAddress common.Address

	// ToAddress refers to the address which was called by the sender.
	ToAddress common.Address

	// ToContractName refers to the name of the contract which was resolved for the ToAddress.
	ToContractName string

	// ToContractAbi refers to the ABI of the contract which was resolved for the ToAddress.
	ToContractAbi *abi.ABI

	// ToInitBytecode refers to the init bytecode recorded for the ToAddress. This is only set if it was being deployed.
	ToInitBytecode []byte

	// ToRuntimeBytecode refers to the bytecode recorded for the ToAddress. This is only set if the contract was
	// successfully deployed in a previous call or at the end of the current call scope.
	ToRuntimeBytecode []byte

	// CodeAddress refers to the address of the code being executed. This can be different from ToAddress if
	// a delegate call was made.
	CodeAddress common.Address

	// CodeContractName refers to the name of the contract which was resolved for the CodeAddress.
	CodeContractName string

	// CodeContractAbi refers to the ABI of the contract which was resolved for the CodeAddress.
	CodeContractAbi *abi.ABI

	// CodeRuntimeBytecode refers to the bytecode recorded for the CodeAddress.
	CodeRuntimeBytecode []byte

	// Operations contains a chronological history of updates in the call frame.
	// Potential types currently are *types.Log (events) or CallFrame (entering of a new child frame).
	Operations []any

	// SelfDestructed indicates whether the call frame executed a SELFDESTRUCT operation.
	SelfDestructed bool

	// InputData refers to the message data the EVM call was made with.
	InputData []byte

	// ConstructorArgsData refers to the subset of InputData that represents constructor argument ABI data. This
	// is only set if this call frame is performing a contract creation. Otherwise, this buffer is always nil.
	ConstructorArgsData []byte

	// ReturnData refers to the data returned by this current call frame.
	ReturnData []byte

	// CallValue describes the ETH value attached to a given CallFrame
	CallValue *big.Int

	// ExecutedCode is a boolean that indicates whether code was executed within a CallFrame. A simple transfer of ETH
	// would be an example of a CallFrame where ExecutedCode would be false
	ExecutedCode bool

	// ReturnError refers to any error returned by the EVM in the current call frame.
	ReturnError error

	// ParentCallFrame refers to the call frame which entered this call frame directly. It may be nil if the current
	// call frame is a top level call frame.
	ParentCallFrame *CallFrame
}

// IsContractCreation indicates whether a contract creation operation was attempted immediately within this call frame.
// This does not include child or parent frames.
// Returns true if this call frame attempted contract creation.
func (c *CallFrame) IsContractCreation() bool {
	return c.ToInitBytecode != nil
}

// IsProxyCall indicates whether the address the message was sent to, and the address the code is being executed from
// are different. This would be indicative of a delegate call.
// Returns true if the code address and to address do not match, implying a delegate call occurred.
func (c *CallFrame) IsProxyCall() bool {
	return c.ToAddress != c.CodeAddress
}

// ChildCallFrames is a getter function that returns all children of the current CallFrame. A child CallFrame is one
// that is entered by this CallFrame
func (c *CallFrame) ChildCallFrames() CallFrames {
	childCallFrames := make(CallFrames, 0)

	// Parse through the Operations array and grab every operation that is of type CallFrame
	for _, operation := range c.Operations {
		if childCallFrame, ok := operation.(*CallFrame); ok {
			childCallFrames = append(childCallFrames, childCallFrame)
		}
	}

	return childCallFrames
}
