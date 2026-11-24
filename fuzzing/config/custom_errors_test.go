package config

import (
	"testing"

	"github.com/crytic/medusa-geth/crypto"
)

// TestParseCustomErrorSelector will test the parsing of custom error signatures and selectors into 4-byte selectors
func TestParseCustomErrorSelector(t *testing.T) {
	// Create the list of valid test cases with known selectors
	validTestCases := []struct {
		input            string
		expectedSelector [4]byte
	}{
		{"Unauthorized()", [4]byte{0x82, 0xb4, 0x29, 0x00}},
		{"Insolvent(uint256)", [4]byte{0x70, 0xcb, 0x15, 0xf5}},
		{"Insolvent(uint)", [4]byte{0x70, 0xcb, 0x15, 0xf5}},
		{"0x70cb15f5", [4]byte{0x70, 0xcb, 0x15, 0xf5}},
		{"0X82B42900", [4]byte{0x82, 0xb4, 0x29, 0x00}},
	}

	// Run each valid test case
	for _, testCase := range validTestCases {
		selector, err := ParseCustomErrorSelector(testCase.input)
		if err != nil {
			t.Errorf("unexpected error parsing %q: %v", testCase.input, err)
			continue
		}
		if selector != testCase.expectedSelector {
			t.Errorf("selector mismatch for %q: expected %x, got %x", testCase.input, testCase.expectedSelector, selector)
		}
	}

	// Create the list of invalid test cases, which should all fail to parse
	invalidTestCases := []string{
		"",
		"Insolvent",
		"Insolvent(uint256 value)",
		"Insolvent( uint256)",
		"1Insolvent()",
		"Insolvent(uint256,)",
		"Insolvent(,uint256)",
		"Insolvent(uint257)",
		"Insolvent(bytes33)",
		"Insolvent(uint7)",
		"Insolvent(int0)",
		"Insolvent(bytes0)",
		"Insolvent(uint256x18)",
		"Insolvent(fixed128x0)",
		"Insolvent(fixed128)",
		"Insolvent(MyStruct)",
		"Insolvent(uint256[0])",
		"Insolvent(uint256[)",
		"Insolvent(())",
		"Insolvent((uint256)",
		"Insolvent(uint256))",
		"Insolvent((uint256,address)x)",
		"0x70cb15",
		"0x70cb15f5aa",
		"0xzzzzzzzz",
	}

	// Run each invalid test case
	for _, input := range invalidTestCases {
		if _, err := ParseCustomErrorSelector(input); err == nil {
			t.Errorf("expected an error parsing %q, but got none", input)
		}
	}
}

// TestCanonicalizeCustomErrorSignature will test that custom error signatures using type aliases are canonicalized,
// including aliases nested within arrays and tuples, and that the resulting selector is computed from the canonical form.
func TestCanonicalizeCustomErrorSignature(t *testing.T) {
	// Create the list of test cases
	testCases := []struct {
		input     string
		canonical string
	}{
		{"E()", "E()"},
		{"E(uint)", "E(uint256)"},
		{"E(int,byte)", "E(int256,bytes1)"},
		{"E(fixed,ufixed)", "E(fixed128x18,ufixed128x18)"},
		{"E(uint[])", "E(uint256[])"},
		{"E(int[3][])", "E(int256[3][])"},
		{"E((uint,address))", "E((uint256,address))"},
		{"E((uint,(int[],byte))[2],bool)", "E((uint256,(int256[],bytes1))[2],bool)"},
		{"E(uint8,bytes32,string,bytes,address[])", "E(uint8,bytes32,string,bytes,address[])"},
		{"E(fixed64x10,function)", "E(fixed64x10,function)"},
	}

	// Run each test case
	for _, testCase := range testCases {
		canonical, err := canonicalizeCustomErrorSignature(testCase.input)
		if err != nil {
			t.Errorf("unexpected error canonicalizing %q: %v", testCase.input, err)
			continue
		}
		if canonical != testCase.canonical {
			t.Errorf("canonical signature mismatch for %q: expected %q, got %q", testCase.input, testCase.canonical, canonical)
		}

		// Verify the selector is derived from the canonical form of the signature.
		selector, err := ParseCustomErrorSelector(testCase.input)
		if err != nil {
			t.Errorf("unexpected error parsing %q: %v", testCase.input, err)
			continue
		}
		var expectedSelector [4]byte
		copy(expectedSelector[:], crypto.Keccak256([]byte(testCase.canonical))[:4])
		if selector != expectedSelector {
			t.Errorf("selector mismatch for %q: expected %x, got %x", testCase.input, expectedSelector, selector)
		}
	}
}
