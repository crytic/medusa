package executiontracer

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/trailofbits/medusa/fuzzing/contracts"
)

// CallFrames represents a list of call frames recorded by the ExecutionTracer.
type CallFrames []*CallFrame

// CallFrame contains information on each EVM call scope, as recorded by an ExecutionTracer.
type CallFrame struct {
	// SenderAddress refers to the address which produced this call.
	SenderAddress common.Address

	// ToAddress refers to the address which was called by the sender.
	ToAddress common.Address

	// ToContract refers to the contract definition which was resolved for the ToAddress.
	ToContract *contracts.Contract

	// ToInitBytecode refers to the init bytecode recorded for the ToAddress. This is only set if it was being deployed.
	ToInitBytecode []byte

	// ToRuntimeBytecode refers to the bytecode recorded for the ToAddress. This is only set if the contract was
	// successfully deployed in a previous call or at the end of the current call scope.
	ToRuntimeBytecode []byte

	// CodeAddress refers to the address of the code being executed. This can be different from ToAddress if
	// a delegate call was made.
	CodeAddress common.Address

	// CodeContract refers to the contract definition which was resolved for the CodeAddress.
	CodeContract *contracts.Contract

	// CodeRuntimeBytecode refers to the bytecode recorded for the CodeAddress.
	CodeRuntimeBytecode []byte

	// Operations contains a chronological history of updates in the call frame.
	// Potential types currently are *types.Log (events) or CallFrame (entering of a new child frame).
	Operations []any

	// SelfDestructed indicates whether the call frame executed a SELFDESTRUCT operation.
	SelfDestructed bool

	// InputData refers to the message data the EVM call was made with.
	InputData []byte

	// ReturnData refers to the data returned by this current call frame.
	ReturnData []byte

	// ReturnError refers to any error returned by the EVM in the current call frame.
	ReturnError error

	// ChildCallFrames refers to any call frames entered by this call frame directly.
	ChildCallFrames CallFrames

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
