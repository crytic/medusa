package utils

import (
	"fmt"
	"strings"

	"github.com/crytic/medusa-geth/accounts/abi"
	compilationTypes "github.com/crytic/medusa/compilation/types"
)

// IsOptimizationTest checks whether the method is an optimization test given potential naming prefixes it must conform to
// and its underlying input/output arguments.
// Returns (isValid, warningMessage). If the method has a prefix but invalid signature, warningMessage explains why.
func IsOptimizationTest(method abi.Method, prefixes []string) (bool, string) {
	// Loop through all enabled prefixes to find a match
	for _, prefix := range prefixes {
		if strings.HasPrefix(method.Name, prefix) {
			// An optimization test must take no inputs and return an int256
			if len(method.Inputs) == 0 && len(method.Outputs) == 1 && method.Outputs[0].Type.T == abi.IntTy && method.Outputs[0].Type.Size == 256 {
				return true, ""
			}
			// Has prefix but invalid signature
			expectedSig := method.Name + "() " + method.StateMutability + " returns (int256)"
			return false, fmt.Sprintf("has signature '%s' but optimization testing provider expects '%s'", method.String(), expectedSig)
		}
	}
	return false, ""
}

// IsPropertyTest checks whether the method is a property test given potential naming prefixes it must conform to
// and its underlying input/output arguments.
// Returns (isValid, warningMessage). If the method has a prefix but invalid signature, warningMessage explains why.
func IsPropertyTest(method abi.Method, prefixes []string) (bool, string) {
	// Loop through all enabled prefixes to find a match
	for _, prefix := range prefixes {
		// The property test must simply have the right prefix and take no inputs and return a boolean
		if strings.HasPrefix(method.Name, prefix) {
			if len(method.Inputs) == 0 && len(method.Outputs) == 1 && method.Outputs[0].Type.T == abi.BoolTy {
				return true, ""
			}
			// Has prefix but invalid signature
			expectedSig := method.Name + "() " + method.StateMutability + " returns (bool)"
			return false, fmt.Sprintf("has signature '%s' but property testing provider expects '%s'", method.String(), expectedSig)
		}
	}
	return false, ""
}

// BinTestByType sorts a contract's methods by whether they are assertion, property, or optimization tests.
// Returns lists of methods for each test type, plus any warnings for methods with test prefixes but invalid signatures.
func BinTestByType(contract *compilationTypes.CompiledContract, propertyTestPrefixes, optimizationTestPrefixes []string, testViewMethods bool) (assertionTests, propertyTests, optimizationTests []abi.Method, warnings []string) {
	warnings = []string{}

	for _, method := range contract.Abi.Methods {
		// Check if it's a property test
		isPropertyTest, propertyWarning := IsPropertyTest(method, propertyTestPrefixes)
		if isPropertyTest {
			propertyTests = append(propertyTests, method)
			continue
		}
		if propertyWarning != "" {
			warnings = append(warnings, fmt.Sprintf("method '%s' %s", method.Name, propertyWarning))
		}

		// Check if it's an optimization test
		isOptimizationTest, optimizationWarning := IsOptimizationTest(method, optimizationTestPrefixes)
		if isOptimizationTest {
			optimizationTests = append(optimizationTests, method)
			continue
		}
		if optimizationWarning != "" {
			warnings = append(warnings, fmt.Sprintf("method '%s' %s", method.Name, optimizationWarning))
		}

		// Not a property or optimization test, check if it's an assertion test
		if !method.IsConstant() || testViewMethods {
			assertionTests = append(assertionTests, method)
		}
	}
	return assertionTests, propertyTests, optimizationTests, warnings
}
