package utils

import (
	"github.com/crytic/medusa/compilation/abiutils"
	"math/big"
)

// HasEncounteredAssertionFailure checks if the provided panic code corresponds to an assertion failure.
// It returns true if an assertion failure is encountered, and false otherwise.
func HasEncounteredAssertionFailure(panicCode *big.Int) bool {
	panicCodes := map[uint64]bool{
		abiutils.PanicCodeAssertFailed:                  true,
		abiutils.PanicCodeArithmeticUnderOverflow:       true,
		abiutils.PanicCodeDivideByZero:                  true,
		abiutils.PanicCodeEnumTypeConversionOutOfBounds: true,
		abiutils.PanicCodeIncorrectStorageAccess:        true,
		abiutils.PanicCodePopEmptyArray:                 true,
		abiutils.PanicCodeOutOfBoundsArrayAccess:        true,
		abiutils.PanicCodeAllocateTooMuchMemory:         true,
		abiutils.PanicCodeCallUninitializedVariable:     true,
	}

	return panicCode != nil && panicCodes[panicCode.Uint64()]
}

// isPanicCodeIncluded checks if the given panic code is included in the config byte array.
// It returns true if the panic code exists in the config, otherwise false.
func IsPanicCodeIncluded(panicCode byte, configBytes []byte) bool {
	for _, configPanicCode := range configBytes {
		if panicCode == configPanicCode {
			return true
		}
	}
	return false
}
