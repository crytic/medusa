package utils

import (
	"strings"
	"testing"

	"github.com/crytic/medusa-geth/accounts/abi"
	compilationTypes "github.com/crytic/medusa/compilation/types"
	"github.com/stretchr/testify/assert"
)

// TestIsPropertyTest_ValidSignature tests that valid property tests are correctly identified
func TestIsPropertyTest_ValidSignature(t *testing.T) {
	boolType, _ := abi.NewType("bool", "", nil)
	method := abi.NewMethod(
		"property_validTest",
		"property_validTest",
		abi.Function,
		"",
		false,
		false,
		abi.Arguments{},
		abi.Arguments{{Type: boolType}},
	)

	isValid, warning := IsPropertyTest(method, []string{"property_"})

	assert.True(t, isValid, "Valid property test should be identified as valid")
	assert.Empty(t, warning, "Valid property test should not generate a warning")
}

// TestIsPropertyTest_InvalidReturnType tests that property tests with wrong return type generate warnings
func TestIsPropertyTest_InvalidReturnType(t *testing.T) {
	uint256Type, _ := abi.NewType("uint256", "", nil)
	method := abi.NewMethod(
		"property_wrongReturn",
		"property_wrongReturn",
		abi.Function,
		"",
		false,
		false,
		abi.Arguments{},
		abi.Arguments{{Type: uint256Type}},
	)

	isValid, warning := IsPropertyTest(method, []string{"property_"})

	assert.False(t, isValid, "Property test with wrong return type should be invalid")
	assert.NotEmpty(t, warning, "Should generate a warning")
	assert.Contains(t, warning, "property_wrongReturn() returns(uint256)")
	assert.Contains(t, warning, "property_wrongReturn()  returns (bool)")
}

// TestIsPropertyTest_HasInput tests that property tests with input parameters generate warnings
func TestIsPropertyTest_HasInput(t *testing.T) {
	uint256Type, _ := abi.NewType("uint256", "", nil)
	boolType, _ := abi.NewType("bool", "", nil)
	method := abi.NewMethod(
		"property_hasInput",
		"property_hasInput",
		abi.Function,
		"",
		false,
		false,
		abi.Arguments{{Type: uint256Type}},
		abi.Arguments{{Type: boolType}},
	)

	isValid, warning := IsPropertyTest(method, []string{"property_"})

	assert.False(t, isValid, "Property test with inputs should be invalid")
	assert.NotEmpty(t, warning, "Should generate a warning")
	assert.Contains(t, warning, "property_hasInput()  returns (bool)")
}

// TestIsPropertyTest_NoPrefix tests that methods without property prefix are not validated
func TestIsPropertyTest_NoPrefix(t *testing.T) {
	uint256Type, _ := abi.NewType("uint256", "", nil)
	method := abi.NewMethod(
		"regularFunction",
		"regularFunction",
		abi.Function,
		"",
		false,
		false,
		abi.Arguments{},
		abi.Arguments{{Type: uint256Type}},
	)

	isValid, warning := IsPropertyTest(method, []string{"property_"})

	assert.False(t, isValid, "Method without prefix should not be identified as property test")
	assert.Empty(t, warning, "Method without prefix should not generate a warning")
}

// TestIsOptimizationTest_ValidSignature tests that valid optimization tests are correctly identified
func TestIsOptimizationTest_ValidSignature(t *testing.T) {
	int256Type, _ := abi.NewType("int256", "", nil)
	method := abi.NewMethod(
		"optimize_validTest",
		"optimize_validTest",
		abi.Function,
		"",
		false,
		false,
		abi.Arguments{},
		abi.Arguments{{Type: int256Type}},
	)

	isValid, warning := IsOptimizationTest(method, []string{"optimize_"})

	assert.True(t, isValid, "Valid optimization test should be identified as valid")
	assert.Empty(t, warning, "Valid optimization test should not generate a warning")
}

// TestIsOptimizationTest_InvalidReturnType tests that optimization tests with wrong return type generate warnings
func TestIsOptimizationTest_InvalidReturnType(t *testing.T) {
	boolType, _ := abi.NewType("bool", "", nil)
	method := abi.NewMethod(
		"optimize_wrongReturn",
		"optimize_wrongReturn",
		abi.Function,
		"",
		false,
		false,
		abi.Arguments{},
		abi.Arguments{{Type: boolType}},
	)

	isValid, warning := IsOptimizationTest(method, []string{"optimize_"})

	assert.False(t, isValid, "Optimization test with wrong return type should be invalid")
	assert.NotEmpty(t, warning, "Should generate a warning")
	assert.Contains(t, warning, "optimize_wrongReturn() returns(bool)")
	assert.Contains(t, warning, "optimize_wrongReturn()  returns (int256)")
}

// TestBinTestByType_ValidMethods tests that valid methods are correctly categorized
func TestBinTestByType_ValidMethods(t *testing.T) {
	boolType, _ := abi.NewType("bool", "", nil)
	int256Type, _ := abi.NewType("int256", "", nil)

	contract := &compilationTypes.CompiledContract{
		Abi: abi.ABI{
			Methods: map[string]abi.Method{
				"property_valid": abi.NewMethod(
					"property_valid",
					"property_valid",
					abi.Function,
					"",
					false,
					false,
					abi.Arguments{},
					abi.Arguments{{Type: boolType}},
				),
				"optimize_valid": abi.NewMethod(
					"optimize_valid",
					"optimize_valid",
					abi.Function,
					"",
					false,
					false,
					abi.Arguments{},
					abi.Arguments{{Type: int256Type}},
				),
			},
		},
	}

	assertionTests, propertyTests, optimizationTests, warnings := BinTestByType(
		contract,
		[]string{"property_"},
		[]string{"optimize_"},
		false,
	)

	assert.Len(t, propertyTests, 1, "Should find 1 valid property test")
	assert.Equal(t, "property_valid", propertyTests[0].Name)

	assert.Len(t, optimizationTests, 1, "Should find 1 valid optimization test")
	assert.Equal(t, "optimize_valid", optimizationTests[0].Name)

	assert.Len(t, assertionTests, 0, "Should have no assertion tests")
	assert.Len(t, warnings, 0, "Should have no warnings for valid methods")
}

// TestBinTestByType_InvalidSignatures tests that invalid methods generate appropriate warnings
func TestBinTestByType_InvalidSignatures(t *testing.T) {
	boolType, _ := abi.NewType("bool", "", nil)
	uint256Type, _ := abi.NewType("uint256", "", nil)
	int256Type, _ := abi.NewType("int256", "", nil)

	contract := &compilationTypes.CompiledContract{
		Abi: abi.ABI{
			Methods: map[string]abi.Method{
				"property_wrongReturn": abi.NewMethod(
					"property_wrongReturn",
					"property_wrongReturn",
					abi.Function,
					"",
					false,
					false,
					abi.Arguments{},
					abi.Arguments{{Type: uint256Type}},
				),
				"property_hasInput": abi.NewMethod(
					"property_hasInput",
					"property_hasInput",
					abi.Function,
					"",
					false,
					false,
					abi.Arguments{{Type: uint256Type}},
					abi.Arguments{{Type: boolType}},
				),
				"property_valid": abi.NewMethod(
					"property_valid",
					"property_valid",
					abi.Function,
					"",
					false,
					false,
					abi.Arguments{},
					abi.Arguments{{Type: boolType}},
				),
				"optimize_wrongReturn": abi.NewMethod(
					"optimize_wrongReturn",
					"optimize_wrongReturn",
					abi.Function,
					"",
					false,
					false,
					abi.Arguments{},
					abi.Arguments{{Type: boolType}},
				),
				"optimize_valid": abi.NewMethod(
					"optimize_valid",
					"optimize_valid",
					abi.Function,
					"",
					false,
					false,
					abi.Arguments{},
					abi.Arguments{{Type: int256Type}},
				),
			},
		},
	}

	assertionTests, propertyTests, optimizationTests, warnings := BinTestByType(
		contract,
		[]string{"property_"},
		[]string{"optimize_"},
		false,
	)

	// Valid methods should be categorized correctly
	assert.Len(t, propertyTests, 1, "Should find 1 valid property test")
	assert.Equal(t, "property_valid", propertyTests[0].Name)

	assert.Len(t, optimizationTests, 1, "Should find 1 valid optimization test")
	assert.Equal(t, "optimize_valid", optimizationTests[0].Name)

	assert.Len(t, assertionTests, 3, "Should have 3 assertion tests since invalid ones fallback to assertion")

	// Should have warnings for invalid methods
	assert.Len(t, warnings, 3, "Should have 3 warnings for invalid methods")

	// Check that warnings contain expected information
	warningMessages := make(map[string]string)
	for _, warning := range warnings {
		// Use strings.Contains for conditional checks (not assert.Contains)
		if strings.Contains(warning, "property_wrongReturn") {
			warningMessages["property_wrongReturn"] = warning
		} else if strings.Contains(warning, "property_hasInput") {
			warningMessages["property_hasInput"] = warning
		} else if strings.Contains(warning, "optimize_wrongReturn") {
			warningMessages["optimize_wrongReturn"] = warning
		}
	}

	// Verify we found all expected warnings
	assert.Len(t, warningMessages, 3, "Should have warnings for all 3 invalid methods")

	// Verify specific warning messages using assert.Contains for actual assertions
	assert.Contains(t, warningMessages["property_wrongReturn"], "property_wrongReturn()  returns (bool)")
	assert.Contains(t, warningMessages["property_hasInput"], "property_hasInput()  returns (bool)")
	assert.Contains(t, warningMessages["optimize_wrongReturn"], "optimize_wrongReturn()  returns (int256)")
}

// TestBinTestByType_MultiplePropertyPrefixes tests support for multiple property test prefixes
func TestBinTestByType_MultiplePropertyPrefixes(t *testing.T) {
	boolType, _ := abi.NewType("bool", "", nil)
	uint256Type, _ := abi.NewType("uint256", "", nil)

	contract := &compilationTypes.CompiledContract{
		Abi: abi.ABI{
			Methods: map[string]abi.Method{
				"property_test1": abi.NewMethod(
					"property_test1",
					"property_test1",
					abi.Function,
					"",
					false,
					false,
					abi.Arguments{},
					abi.Arguments{{Type: boolType}},
				),
				"invariant_test1": abi.NewMethod(
					"invariant_test1",
					"invariant_test1",
					abi.Function,
					"",
					false,
					false,
					abi.Arguments{},
					abi.Arguments{{Type: boolType}},
				),
				"invariant_invalid": abi.NewMethod(
					"invariant_invalid",
					"invariant_invalid",
					abi.Function,
					"",
					false,
					false,
					abi.Arguments{{Type: uint256Type}},
					abi.Arguments{{Type: boolType}},
				),
			},
		},
	}

	assertionTests, propertyTests, optimizationTests, warnings := BinTestByType(
		contract,
		[]string{"property_", "invariant_"},
		[]string{},
		false,
	)

	assert.Len(t, propertyTests, 2, "Should find 2 valid property tests with different prefixes")
	assert.Len(t, optimizationTests, 0, "Should have no optimization tests")
	assert.Len(t, assertionTests, 1, "Should have no assertion tests")
	assert.Len(t, warnings, 1, "Should have 1 warning for invalid invariant test")
	assert.Contains(t, warnings[0], "invariant_invalid")
}
