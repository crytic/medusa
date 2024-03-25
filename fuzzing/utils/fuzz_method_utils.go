package utils

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"strings"
)

// IsOptimizationTest checks whether the method is an optimization test given potential naming prefixes it must conform to
// and its underlying input/output arguments.
func IsOptimizationTest(method abi.Method, prefixes []string) bool {
	// Loop through all enabled prefixes to find a match
	for _, prefix := range prefixes {
		if strings.HasPrefix(method.Name, prefix) {
			// An optimization test must take no inputs and return an int256
			if len(method.Inputs) == 0 && len(method.Outputs) == 1 && method.Outputs[0].Type.T == abi.IntTy && method.Outputs[0].Type.Size == 256 {
				return true
			}
		}
	}
	return false
}

// IsPropertyTest checks whether the method is a property test given potential naming prefixes it must conform to
// and its underlying input/output arguments.
func IsPropertyTest(method abi.Method, prefixes []string) bool {
	// Loop through all enabled prefixes to find a match
	for _, prefix := range prefixes {
		// The property test must simply have the right prefix and take no inputs
		if strings.HasPrefix(method.Name, prefix) && len(method.Inputs) == 0 {
			return true
		}
	}
	return false
}
