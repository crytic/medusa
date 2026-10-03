package types

import (
	"testing"
)

func TestIsTestContract(t *testing.T) {
	tests := []struct {
		name           string
		contractName   string
		expectedResult bool
	}{
		{"Contract with Test suffix", "MyTest", true},
		{"Contract with Test prefix", "TestContract", true},
		{"Contract with test in middle", "MyTestContract", true},
		{"Contract without test", "MyContract", false},
		{"Empty contract name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contract := ContractDefinition{
				CanonicalName: tt.contractName,
			}
			result := IsTestContract(contract)
			if result != tt.expectedResult {
				t.Errorf("IsTestContract(%s) = %v, want %v", tt.contractName, result, tt.expectedResult)
			}
		})
	}
}

func TestIsTestFunction(t *testing.T) {
	tests := []struct {
		name           string
		functionName   string
		expectedResult bool
	}{
		{"Function with test prefix", "testTransfer", true},
		{"Function with Test prefix", "TestTransfer", true},
		{"Function with invariant prefix", "invariant_balance", true},
		{"Function with testfuzz prefix", "testfuzz_deposit", true},
		{"Function without test prefix", "transfer", false},
		{"Empty function name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			function := FunctionDefinition{
				Name: tt.functionName,
			}
			result := IsTestFunction(function)
			if result != tt.expectedResult {
				t.Errorf("IsTestFunction(%s) = %v, want %v", tt.functionName, result, tt.expectedResult)
			}
		})
	}
}

func TestIsBuiltinFunction(t *testing.T) {
	tests := []struct {
		name           string
		functionName   string
		expectedResult bool
	}{
		{"require is builtin", "require", true},
		{"assert is builtin", "assert", true},
		{"revert is builtin", "revert", true},
		{"keccak256 is builtin", "keccak256", true},
		{"transfer is not builtin", "transfer", false},
		{"deposit is not builtin", "deposit", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBuiltinFunction(tt.functionName)
			if result != tt.expectedResult {
				t.Errorf("isBuiltinFunction(%s) = %v, want %v", tt.functionName, result, tt.expectedResult)
			}
		})
	}
}

func TestExtractContractName(t *testing.T) {
	tests := []struct {
		name           string
		jsonData       string
		expectedResult string
	}{
		{
			"Simple identifier",
			`{"nodeType": "Identifier", "name": "myContract"}`,
			"myContract",
		},
		{
			"Empty identifier",
			`{"nodeType": "Identifier", "name": ""}`,
			"",
		},
		{
			"Non-identifier node",
			`{"nodeType": "Literal"}`,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractContractName([]byte(tt.jsonData))
			if result != tt.expectedResult {
				t.Errorf("extractContractName() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}
