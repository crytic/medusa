package abiutils

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/vm"
)

// An enum is defined below providing all `Panic(uint)` error codes returned in return data when the VM encounters
// an error in some cases.
// Reference: https://docs.soliditylang.org/en/latest/control-structures.html#panic-via-assert-and-error-via-require
const (
	PanicCodeCompilerInserted              = 0x00
	PanicCodeAssertFailed                  = 0x01
	PanicCodeArithmeticUnderOverflow       = 0x11
	PanicCodeDivideByZero                  = 0x12
	PanicCodeEnumTypeConversionOutOfBounds = 0x21
	PanicCodeIncorrectStorageAccess        = 0x22
	PanicCodePopEmptyArray                 = 0x31
	PanicCodeOutOfBoundsArrayAccess        = 0x32
	PanicCodeAllocateTooMuchMemory         = 0x41
	PanicCodeCallUninitializedVariable     = 0x51
)

// GetSolidityPanicCode obtains a panic code from a VM error and return data, if possible.
// A flag is provided indicating whether assertion failures in older Solidity compilations will be also mapped onto
// newer Solidity panic code.
// If the error and return data are not representative of a Panic, then nil is returned.
func GetSolidityPanicCode(returnError error, returnData []byte, backwardsCompatible bool) *big.Int {
	// If this method is backwards compatible with older solidity, there is no panic code, and we simply look
	// for a specific error that used to represent assertion failures.
	_, hitInvalidOpcode := returnError.(*vm.ErrInvalidOpCode)
	if backwardsCompatible && hitInvalidOpcode {
		return big.NewInt(PanicCodeAssertFailed)
	}

	// Verify we have a revert, and our return data fits exactly the selector + uint256
	if errors.Is(returnError, vm.ErrExecutionReverted) && len(returnData) == 4+32 {
		uintType, _ := abi.NewType("uint256", "", nil)
		panicReturnDataAbi := abi.NewMethod("Panic", "Panic", abi.Function, "", false, false, []abi.Argument{
			{Name: "", Type: uintType, Indexed: false},
		}, abi.Arguments{})

		// Verify the return data starts with the correct selector, then unpack the arguments.
		if bytes.Equal(returnData[:4], panicReturnDataAbi.ID) {
			values, err := panicReturnDataAbi.Inputs.Unpack(returnData[4:])

			// If they unpacked without issue, read the panic code.
			if err == nil && len(values) > 0 {
				panicCode := values[0].(*big.Int)
				return panicCode
			}
		}
	}
	return nil
}

// GetSolidityRevertErrorString obtains an error message from a VM error and return data, if possible.
// If the error and return data are not representative of an Error, then nil is returned.
func GetSolidityRevertErrorString(returnError error, returnData []byte) *string {
	// Verify we have a revert, and our return data fits the selector + additional data.
	if errors.Is(returnError, vm.ErrExecutionReverted) && len(returnData) > 4 {
		stringType, _ := abi.NewType("string", "", nil)
		errorReturnDataAbi := abi.NewMethod("Error", "Error", abi.Function, "", false, false, []abi.Argument{
			{Name: "", Type: stringType, Indexed: false},
		}, abi.Arguments{})

		// Verify the return data starts with the correct selector, then unpack the arguments.
		if bytes.Equal(returnData[:4], errorReturnDataAbi.ID) {
			values, err := errorReturnDataAbi.Inputs.Unpack(returnData[4:])

			// If they unpacked without issue, read the error string.
			if err == nil && len(values) > 0 {
				errorMessage := values[0].(string)
				return &errorMessage
			}
		}
	}

	return nil
}

// GetSolidityCustomRevertError obtains a custom Solidity error returned, if one was and could be resolved.
// Returns the ABI error definition as well as its unpacked values. Or returns nil outputs if a custom error was not
// emitted, or could not be resolved.
func GetSolidityCustomRevertError(contractAbi *abi.ABI, returnError error, returnData []byte) (*abi.Error, []any) {
	// If no ABI was given or a revert was not encountered, no custom error can be extracted, or may exist,
	// respectively.
	if !errors.Is(returnError, vm.ErrExecutionReverted) || contractAbi == nil {
		return nil, nil
	}

	// Loop for each error definition in the ABI.
	for _, abiError := range contractAbi.Errors {
		// If the data's leading selector value matches the ID of the error, return it.
		if len(returnData) >= 4 && bytes.Equal(abiError.ID.Bytes()[:4], returnData[:4]) {
			// Make a local copy to avoid taking a pointer of a loop variable and having a memory leak
			matchedCustomError := &abiError
			unpackedCustomErrorArgs, err := matchedCustomError.Inputs.Unpack(returnData[4:])
			if err == nil {
				return matchedCustomError, unpackedCustomErrorArgs
			}
		}
	}
	return nil, nil
}

// GetPanicReason will take in a panic code as an uint64 and will return the string reason behind that panic code. For
// example, if panic code is PanicCodeAssertFailed, then "assertion failure" is returned.
func GetPanicReason(panicCode uint64) string {
	// Switch on panic code
	switch panicCode {
	case PanicCodeCompilerInserted:
		return "panic: compiler inserted panic"
	case PanicCodeAssertFailed:
		return "panic: assertion failed"
	case PanicCodeArithmeticUnderOverflow:
		return "panic: arithmetic underflow"
	case PanicCodeDivideByZero:
		return "panic: division by zero"
	case PanicCodeEnumTypeConversionOutOfBounds:
		return "panic: enum access out of bounds"
	case PanicCodeIncorrectStorageAccess:
		return "panic: incorrect storage access"
	case PanicCodePopEmptyArray:
		return "panic: pop on empty array"
	case PanicCodeOutOfBoundsArrayAccess:
		return "panic: out of bounds array access"
	case PanicCodeAllocateTooMuchMemory:
		return "panic; overallocation of memory"
	case PanicCodeCallUninitializedVariable:
		return "panic: call on uninitialized variable"
	default:
		return fmt.Sprintf("unknown panic code(%v)", panicCode)
	}
}
