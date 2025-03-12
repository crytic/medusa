package config

import (
	"encoding/json"
	"math/big"
	"testing"
)

// TestUnmarshalBalances will test the unmarshalling of a ContractBalance from a string
func TestUnmarshalBalances(t *testing.T) {
	// Create the list of test cases
	testCases := []struct {
		input           string
		expectedBalance ContractBalance
	}{
		{"\"\"", ContractBalance{*big.NewInt(0)}},
		{"\"1\"", ContractBalance{*big.NewInt(1)}},
		{"\"100\"", ContractBalance{*big.NewInt(100)}},
		{"\"0\"", ContractBalance{*big.NewInt(0)}},
		{"\"1e5\"", ContractBalance{*big.NewInt(100000)}},
		{"\"10E-1\"", ContractBalance{*big.NewInt(1)}},
		{"\"0x1337\"", ContractBalance{*big.NewInt(4919)}},
		{"\"0X10\"", ContractBalance{*big.NewInt(16)}},
	}

	// Iterate over the test cases and unmarshal the input string into a ContractBalance
	for _, tc := range testCases {
		var b ContractBalance
		if err := json.Unmarshal([]byte(tc.input), &b); err != nil {
			t.Errorf("Unmarshal(%q): unexpected error: %v", tc.input, err)
		}
		if b.Cmp(&tc.expectedBalance.Int) != 0 {
			t.Errorf("Unmarshal(%q) = %v, want %v", tc.input, b, tc.expectedBalance)
		}
	}
}

// TestMarshalBalances will test the marshalling of a ContractBalance to a string
func TestMarshalBalances(t *testing.T) {
	// Create the list of test cases
	testCases := []struct {
		input           ContractBalance
		expectedBalance string
	}{
		{ContractBalance{*big.NewInt(0)}, "\"0\""},
		{ContractBalance{*big.NewInt(1)}, "\"1\""},
		{ContractBalance{*big.NewInt(0).Mul(big.NewInt(1000000000000000000), big.NewInt(1000000000000000000))}, "\"1000000000000000000000000000000000000\""},
	}

	// Iterate over the test cases and marshal the ContractBalance to a string
	for _, tc := range testCases {
		out, err := json.Marshal(tc.input)
		if err != nil {
			t.Errorf("Marshal(%v): unexpected error: %v", tc.input, err)
		}
		if string(out) != tc.expectedBalance {
			t.Errorf("Marshal(%v) = %v, want %v", tc.input, out, tc.expectedBalance)
		}
	}
}
