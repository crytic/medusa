package config

import (
	"encoding/json"
	"math/big"
	"testing"
)

func TestDeserializeBalances(t *testing.T) {
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
