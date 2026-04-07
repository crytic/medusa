package state

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/crytic/medusa-geth/common"
	gethTypes "github.com/crytic/medusa-geth/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadGenesisAllocFromFile(t *testing.T) {
	t.Parallel()

	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	t.Run("ValidForgeFormat", func(t *testing.T) {
		// Create a test file with forge/anvil format
		testFile := filepath.Join(tmpDir, "test_state.json")
		testData := map[string]interface{}{
			"0x1111111111111111111111111111111111111111": map[string]interface{}{
				"balance": "0x56bc75e2d63100000", // 100 ETH in wei
				"nonce":   "0x5",
				"code":    "0x6080604052",
				"storage": map[string]string{
					"0x0000000000000000000000000000000000000000000000000000000000000001": "0x0000000000000000000000000000000000000000000000000000000000000002",
				},
			},
			"0x2222222222222222222222222222222222222222": map[string]interface{}{
				"balance": "0xde0b6b3a7640000", // 1 ETH in wei
				"nonce":   "0x0",
			},
		}

		jsonData, err := json.Marshal(testData)
		require.NoError(t, err)
		err = os.WriteFile(testFile, jsonData, 0644)
		require.NoError(t, err)

		// Load the file
		genesisAlloc, err := LoadGenesisAllocFromFile(testFile)
		require.NoError(t, err)

		// Verify the results
		assert.Len(t, genesisAlloc, 2)

		// Check first account
		addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
		account1, exists := genesisAlloc[addr1]
		require.True(t, exists)

		expectedBalance1, _ := new(big.Int).SetString("100000000000000000000", 10) // 100 ETH
		assert.Equal(t, expectedBalance1, account1.Balance)
		assert.Equal(t, uint64(5), account1.Nonce)
		assert.Equal(t, []byte{0x60, 0x80, 0x60, 0x40, 0x52}, account1.Code)
		assert.Len(t, account1.Storage, 1)

		// Check storage
		storageKey := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")
		storageValue := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002")
		assert.Equal(t, storageValue, account1.Storage[storageKey])

		// Check second account
		addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
		account2, exists := genesisAlloc[addr2]
		require.True(t, exists)

		expectedBalance2, _ := new(big.Int).SetString("1000000000000000000", 10) // 1 ETH
		assert.Equal(t, expectedBalance2, account2.Balance)
		assert.Equal(t, uint64(0), account2.Nonce)
		assert.Empty(t, account2.Code)
		assert.Empty(t, account2.Storage)
	})

	t.Run("EmptyFile", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "empty_state.json")
		err := os.WriteFile(testFile, []byte("{}"), 0644)
		require.NoError(t, err)

		genesisAlloc, err := LoadGenesisAllocFromFile(testFile)
		require.NoError(t, err)
		assert.Len(t, genesisAlloc, 0)
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		_, err := LoadGenesisAllocFromFile(filepath.Join(tmpDir, "nonexistent.json"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read genesis state file")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "invalid.json")
		err := os.WriteFile(testFile, []byte("{invalid json}"), 0644)
		require.NoError(t, err)

		_, err = LoadGenesisAllocFromFile(testFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to detect genesis format")
	})
}

func TestMergeGenesisAllocs(t *testing.T) {
	t.Parallel()

	t.Run("MergeTwoAllocations", func(t *testing.T) {
		// Create base allocation
		baseAlloc := gethTypes.GenesisAlloc{
			common.HexToAddress("0x1111111111111111111111111111111111111111"): {
				Balance: big.NewInt(100),
				Nonce:   1,
				Code:    []byte{0x01, 0x02},
				Storage: map[common.Hash]common.Hash{
					common.HexToHash("0x01"): common.HexToHash("0x02"),
				},
			},
			common.HexToAddress("0x2222222222222222222222222222222222222222"): {
				Balance: big.NewInt(200),
				Nonce:   2,
			},
		}

		// Create priority allocation that overlaps with one account
		priorityAlloc := gethTypes.GenesisAlloc{
			common.HexToAddress("0x1111111111111111111111111111111111111111"): {
				Balance: big.NewInt(999),
				Nonce:   99,
				Code:    []byte{0x03, 0x04},
				Storage: map[common.Hash]common.Hash{
					common.HexToHash("0x03"): common.HexToHash("0x04"),
				},
			},
			common.HexToAddress("0x3333333333333333333333333333333333333333"): {
				Balance: big.NewInt(300),
				Nonce:   3,
			},
		}

		// Merge with priority taking precedence
		result := MergeGenesisAllocs(priorityAlloc, baseAlloc)

		// Should have 3 accounts total
		assert.Len(t, result, 3)

		// Account 0x1111... should have priority values
		addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
		assert.Equal(t, big.NewInt(999), result[addr1].Balance)
		assert.Equal(t, uint64(99), result[addr1].Nonce)
		assert.Equal(t, []byte{0x03, 0x04}, result[addr1].Code)
		assert.Len(t, result[addr1].Storage, 1)
		assert.Equal(t, common.HexToHash("0x04"), result[addr1].Storage[common.HexToHash("0x03")])

		// Account 0x2222... should have base values
		addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
		assert.Equal(t, big.NewInt(200), result[addr2].Balance)
		assert.Equal(t, uint64(2), result[addr2].Nonce)

		// Account 0x3333... should have priority values
		addr3 := common.HexToAddress("0x3333333333333333333333333333333333333333")
		assert.Equal(t, big.NewInt(300), result[addr3].Balance)
		assert.Equal(t, uint64(3), result[addr3].Nonce)
	})

	t.Run("EmptyAllocations", func(t *testing.T) {
		result := MergeGenesisAllocs(gethTypes.GenesisAlloc{}, gethTypes.GenesisAlloc{})
		assert.Len(t, result, 0)
	})

	t.Run("DeepCopy", func(t *testing.T) {
		// Test that the merge creates deep copies and doesn't share references
		baseAlloc := gethTypes.GenesisAlloc{
			common.HexToAddress("0x1111111111111111111111111111111111111111"): {
				Balance: big.NewInt(100),
				Storage: map[common.Hash]common.Hash{
					common.HexToHash("0x01"): common.HexToHash("0x02"),
				},
			},
		}

		priorityAlloc := gethTypes.GenesisAlloc{}

		result := MergeGenesisAllocs(priorityAlloc, baseAlloc)

		// Modify the result
		addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
		result[addr].Balance.SetInt64(999)
		result[addr].Storage[common.HexToHash("0x01")] = common.HexToHash("0x99")

		// Original should be unchanged
		assert.Equal(t, int64(100), baseAlloc[addr].Balance.Int64())
		assert.Equal(t, common.HexToHash("0x02"), baseAlloc[addr].Storage[common.HexToHash("0x01")])
	})
}

func TestCopyStorage(t *testing.T) {
	t.Parallel()

	t.Run("NilStorage", func(t *testing.T) {
		result := copyStorage(nil)
		assert.Nil(t, result)
	})

	t.Run("EmptyStorage", func(t *testing.T) {
		result := copyStorage(map[common.Hash]common.Hash{})
		assert.NotNil(t, result)
		assert.Len(t, result, 0)
	})

	t.Run("CopyStorage", func(t *testing.T) {
		original := map[common.Hash]common.Hash{
			common.HexToHash("0x01"): common.HexToHash("0x02"),
			common.HexToHash("0x03"): common.HexToHash("0x04"),
		}

		result := copyStorage(original)

		// Should be equal
		assert.Equal(t, original, result)

		// But not the same reference
		result[common.HexToHash("0x01")] = common.HexToHash("0x99")
		assert.Equal(t, common.HexToHash("0x02"), original[common.HexToHash("0x01")])
	})
}

func TestFlexibleNonce_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected uint64
		wantErr  bool
	}{
		{"integer_zero", `0`, 0, false},
		{"integer_small", `5`, 5, false},
		{"integer_large", `999`, 999, false},
		{"hex_string_zero", `"0x0"`, 0, false},
		{"hex_string_small", `"0x5"`, 5, false},
		{"hex_string_large", `"0x3e7"`, 999, false},
		{"invalid_string", `"not a number"`, 0, true},
		{"invalid_bool", `true`, 0, true},
		{"invalid_null", `null`, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n flexibleNonce
			err := json.Unmarshal([]byte(tt.input), &n)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, uint64(n))
			}
		})
	}
}

func TestDetectGenesisFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected genesisFormat
	}{
		{
			"native_anvil_lowercase",
			`"0x1f8b0800000000..."`,
			formatNativeAnvil,
		},
		{
			"native_anvil_uppercase",
			`"0x1F8B0800000000..."`,
			formatNativeAnvil,
		},
		{
			"anvil_wrapper",
			`{"block": {}, "accounts": {"0x1111...": {}}}`,
			formatAnvilWrapper,
		},
		{
			"plain_accounts",
			`{"0x1111111111111111111111111111111111111111": {"balance": "0x0", "nonce": "0x0"}}`,
			formatPlainAccounts,
		},
		{
			"empty_accounts",
			`{}`,
			formatPlainAccounts,
		},
		{
			"invalid_array",
			`[]`,
			formatUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectGenesisFormat([]byte(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

// makeNativeAnvilDump synthesizes a native anvil_dumpState file (gzip-compressed JSON hex string)
// from a plain accounts map, matching the format produced by `cast rpc anvil_dumpState`.
func makeNativeAnvilDump(t *testing.T, accounts map[string]interface{}) []byte {
	t.Helper()

	wrapper := map[string]interface{}{"accounts": accounts}
	inner, err := json.Marshal(wrapper)
	require.NoError(t, err)

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err = gz.Write(inner)
	require.NoError(t, err)
	require.NoError(t, gz.Close())

	// Encode as a JSON-quoted hex string with 0x prefix, matching anvil output.
	hexStr := fmt.Sprintf(`"0x%s"`, hex.EncodeToString(buf.Bytes()))
	return []byte(hexStr)
}

func TestLoadGenesisAllocFromFile_NativeAnvilFormat(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "anvil_state.json")

	// Build a synthetic native anvil dump with one account, using integer nonce
	// (the format anvil_dumpState actually produces).
	accounts := map[string]interface{}{
		"0x1111111111111111111111111111111111111111": map[string]interface{}{
			"balance": "0x56bc75e2d63100000", // 100 ETH
			"nonce":   1,                     // integer nonce (native anvil format)
			"code":    "0x6080604052",
			"storage": map[string]string{
				"0x0000000000000000000000000000000000000000000000000000000000000001": "0x0000000000000000000000000000000000000000000000000000000000000002",
			},
		},
	}

	data := makeNativeAnvilDump(t, accounts)
	require.NoError(t, os.WriteFile(testFile, data, 0o644))

	genesisAlloc, err := LoadGenesisAllocFromFile(testFile)
	require.NoError(t, err)
	require.Len(t, genesisAlloc, 1)

	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	account, exists := genesisAlloc[addr]
	require.True(t, exists)

	expected, _ := new(big.Int).SetString("100000000000000000000", 10)
	assert.Equal(t, expected, account.Balance)
	assert.Equal(t, uint64(1), account.Nonce)
	assert.Equal(t, []byte{0x60, 0x80, 0x60, 0x40, 0x52}, account.Code)
	storageKey := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")
	storageVal := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002")
	assert.Equal(t, storageVal, account.Storage[storageKey])
}

func TestLoadGenesisAllocFromFile_IntegerNonces(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "integer_nonce.json")

	// Create test file with integer nonce
	testData := map[string]interface{}{
		"0x1111111111111111111111111111111111111111": map[string]interface{}{
			"balance": "0x56bc75e2d63100000",
			"nonce":   5, // Integer, not hex string
			"code":    "0x6080604052",
			"storage": map[string]string{},
		},
	}

	jsonData, err := json.Marshal(testData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, jsonData, 0644)
	require.NoError(t, err)

	// Load and verify
	genesisAlloc, err := LoadGenesisAllocFromFile(testFile)
	require.NoError(t, err)

	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	account := genesisAlloc[addr]
	assert.Equal(t, uint64(5), account.Nonce)
}

func TestLoadGenesisAllocFromFile_AnvilWrapperFormat(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "wrapper.json")

	// Create test file with anvil wrapper structure
	testData := map[string]interface{}{
		"block": map[string]interface{}{
			"number": "0x4",
		},
		"accounts": map[string]interface{}{
			"0x1111111111111111111111111111111111111111": map[string]interface{}{
				"balance": "0x56bc75e2d63100000",
				"nonce":   0,
				"code":    "0x",
				"storage": map[string]string{},
			},
		},
	}

	jsonData, err := json.Marshal(testData)
	require.NoError(t, err)
	err = os.WriteFile(testFile, jsonData, 0644)
	require.NoError(t, err)

	// Load and verify
	genesisAlloc, err := LoadGenesisAllocFromFile(testFile)
	require.NoError(t, err)
	assert.Len(t, genesisAlloc, 1)
}
