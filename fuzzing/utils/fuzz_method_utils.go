package utils

import (
	"strings"

	compilationTypes "github.com/crytic/medusa/compilation/types"
	"github.com/crytic/medusa/fuzzing/config"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// IsOptimizationTest checks whether the method is an optimization test given potential naming prefixes it must conform to
// and its underlying input/output arguments.
func isOptimizationTest(method abi.Method, prefixes []string) bool {
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
func isPropertyTest(method abi.Method, prefixes []string) bool {
	// Loop through all enabled prefixes to find a match
	for _, prefix := range prefixes {
		// The property test must simply have the right prefix and take no inputs
		if strings.HasPrefix(method.Name, prefix) && len(method.Inputs) == 0 {
			return true
		}
	}
	return false
}

// IsAssertionTest checks whether the method is configured by the attached fuzzer to be a target of assertion testing.
// Returns true if this target should be tested, false otherwise.
func isAssertionTest(method abi.Method, includeViewMethods bool) bool {
	// Only test constant methods (pure/view) if we are configured to.
	return !method.IsConstant() || includeViewMethods
}

// BinTestByType sorts a contract's methods by whether they are assertion, property, or optimization tests.
func BinTestByType(contract *compilationTypes.CompiledContract, testCfg config.TestingConfig) (assertionTests, propertyTests, optimizationTests []abi.Method) {
	for _, method := range contract.Abi.Methods {
		if isPropertyTest(method, testCfg.PropertyTesting.TestPrefixes) {
			propertyTests = append(propertyTests, method)
		} else if isOptimizationTest(method, testCfg.OptimizationTesting.TestPrefixes) {
			optimizationTests = append(optimizationTests, method)
		} else if isAssertionTest(method, testCfg.AssertionTesting.TestViewMethods) {
			assertionTests = append(assertionTests, method)
		}
	}
	return assertionTests, propertyTests, optimizationTests
}
