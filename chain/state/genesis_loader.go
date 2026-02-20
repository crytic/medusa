package state

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/common/hexutil"
	gethTypes "github.com/crytic/medusa-geth/core/types"
	"github.com/holiman/uint256"
)

// flexibleNonce handles both integer (0) and hex string ("0x0") nonce formats
type flexibleNonce uint64

func (n *flexibleNonce) UnmarshalJSON(data []byte) error {
	// Reject null explicitly
	if bytes.Equal(data, []byte("null")) {
		return fmt.Errorf("nonce must be integer or hex string, got: null")
	}

	// Try unmarshaling as integer first (anvil_dumpState format)
	var intVal uint64
	if err := json.Unmarshal(data, &intVal); err == nil {
		*n = flexibleNonce(intVal)
		return nil
	}

	// Fall back to hex string (current format)
	var hexVal hexutil.Uint64
	if err := json.Unmarshal(data, &hexVal); err == nil {
		*n = flexibleNonce(hexVal)
		return nil
	}

	return fmt.Errorf("nonce must be integer or hex string, got: %s", string(data))
}

// forgeAllocAccount represents an account in the forge/anvil state dump format.
// This matches the format output by anvil_dumpState and used by Optimism's op-chain-ops.
type forgeAllocAccount struct {
	Balance hexutil.U256                `json:"balance"`
	Nonce   flexibleNonce               `json:"nonce"`
	Code    hexutil.Bytes               `json:"code,omitempty"`
	Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
}

// anvilDumpStateWrapper represents the full anvil_dumpState structure
type anvilDumpStateWrapper struct {
	Accounts map[common.Address]forgeAllocAccount `json:"accounts"`
}

type genesisFormat int

const (
	formatUnknown       genesisFormat = iota
	formatNativeAnvil                 // Compressed hex string
	formatAnvilWrapper                // Decompressed with wrapper
	formatPlainAccounts               // Current format
)

// detectGenesisFormat examines file content to determine format
func detectGenesisFormat(data []byte) genesisFormat {
	// Trim whitespace
	trimmed := bytes.TrimSpace(data)

	// Check for compressed anvil dump (starts with "0x1f8b08" - gzip magic in hex)
	if bytes.HasPrefix(trimmed, []byte(`"0x1f8b08`)) || bytes.HasPrefix(trimmed, []byte(`"0x1F8B08`)) {
		return formatNativeAnvil
	}

	// Try to parse as JSON to check structure
	var testMap map[string]interface{}
	if err := json.Unmarshal(trimmed, &testMap); err != nil {
		return formatUnknown
	}

	// Check for anvil wrapper (has "accounts" field)
	if _, hasAccounts := testMap["accounts"]; hasAccounts {
		return formatAnvilWrapper
	}

	// Empty object is valid plain accounts format (empty allocation)
	if len(testMap) == 0 {
		return formatPlainAccounts
	}

	// Check if keys look like addresses (0x followed by 40 hex chars)
	for key := range testMap {
		if len(key) == 42 && strings.HasPrefix(key, "0x") {
			return formatPlainAccounts
		}
		// If first key doesn't match, it's not plain accounts format
		break
	}

	return formatUnknown
}

// loadNativeAnvilDumpState handles gzip-compressed anvil_dumpState
func loadNativeAnvilDumpState(data []byte) (gethTypes.GenesisAlloc, error) {
	// Parse the JSON-quoted hex string
	var hexString string
	if err := json.Unmarshal(data, &hexString); err != nil {
		return nil, fmt.Errorf("expected JSON-quoted hex string: %w", err)
	}

	// Remove "0x" prefix
	hexString = strings.TrimPrefix(hexString, "0x")
	hexString = strings.TrimPrefix(hexString, "0X")

	// Decode hex to bytes
	compressed, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, fmt.Errorf("invalid hex data: %w", err)
	}

	// Decompress gzip
	gzipReader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	var decompressed bytes.Buffer
	if _, err := decompressed.ReadFrom(gzipReader); err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}

	// Parse as anvil wrapper
	return loadAnvilWrapper(decompressed.Bytes())
}

// loadAnvilWrapper handles decompressed anvil_dumpState wrapper
func loadAnvilWrapper(data []byte) (gethTypes.GenesisAlloc, error) {
	var wrapper anvilDumpStateWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse anvil wrapper: %w", err)
	}

	if wrapper.Accounts == nil {
		return nil, fmt.Errorf("anvil wrapper missing 'accounts' field")
	}

	return convertForgeAllocsToGenesis(wrapper.Accounts), nil
}

// loadPlainAccounts handles the current plain JSON format
func loadPlainAccounts(data []byte) (gethTypes.GenesisAlloc, error) {
	var forgeAllocs map[common.Address]forgeAllocAccount
	if err := json.Unmarshal(data, &forgeAllocs); err != nil {
		return nil, fmt.Errorf("failed to parse accounts: %w", err)
	}

	return convertForgeAllocsToGenesis(forgeAllocs), nil
}

// convertForgeAllocsToGenesis converts forge format to geth GenesisAlloc
func convertForgeAllocsToGenesis(forgeAllocs map[common.Address]forgeAllocAccount) gethTypes.GenesisAlloc {
	genesisAlloc := make(gethTypes.GenesisAlloc, len(forgeAllocs))
	for addr, forgeAccount := range forgeAllocs {
		u256Int := (*uint256.Int)(&forgeAccount.Balance)
		balance := u256Int.ToBig()
		nonce := uint64(forgeAccount.Nonce)
		code := []byte(forgeAccount.Code)

		storage := make(map[common.Hash]common.Hash, len(forgeAccount.Storage))
		for key, value := range forgeAccount.Storage {
			storage[key] = value
		}

		genesisAlloc[addr] = gethTypes.Account{
			Balance: balance,
			Nonce:   nonce,
			Code:    code,
			Storage: storage,
		}
	}
	return genesisAlloc
}

// LoadGenesisAllocFromFile loads a genesis allocation from various formats:
//
// 1. Native anvil_dumpState: Gzip-compressed hex string (from: cast rpc anvil_dumpState > file.json)
//   - Automatically decompresses and extracts accounts field
//   - Supports both integer and hex-encoded nonces
//
// 2. Decompressed anvil wrapper: Plain JSON with {"block": {...}, "accounts": {...}}
//   - Extracts accounts field automatically
//
// 3. Plain accounts JSON: Direct mapping of addresses to account state (legacy format)
//   - Format: {"0xADDRESS": {"balance": "0x...", "nonce": "0x...", ...}}
//
// The function automatically detects which format is being used.
func LoadGenesisAllocFromFile(filePath string) (gethTypes.GenesisAlloc, error) {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read genesis state file: %w", err)
	}

	// Detect format
	format := detectGenesisFormat(data)

	// Load based on format
	switch format {
	case formatNativeAnvil:
		return loadNativeAnvilDumpState(data)
	case formatAnvilWrapper:
		return loadAnvilWrapper(data)
	case formatPlainAccounts:
		return loadPlainAccounts(data)
	default:
		return nil, fmt.Errorf("failed to detect genesis format: file must be either compressed anvil_dumpState, anvil wrapper JSON, or plain accounts JSON")
	}
}

// MergeGenesisAllocs merges two genesis allocations, with priority given to accounts in the priority allocation.
// If an account exists in both allocations, the priority allocation's values take precedence.
// This is useful for combining a loaded state file with fuzzer-generated accounts.
func MergeGenesisAllocs(priority, base gethTypes.GenesisAlloc) gethTypes.GenesisAlloc {
	result := make(gethTypes.GenesisAlloc, len(base)+len(priority))

	// Copy base allocations
	for addr, account := range base {
		result[addr] = gethTypes.Account{
			Balance: new(big.Int).Set(account.Balance),
			Nonce:   account.Nonce,
			Code:    append([]byte(nil), account.Code...),
			Storage: copyStorage(account.Storage),
		}
	}

	// Override with priority allocations
	for addr, account := range priority {
		result[addr] = gethTypes.Account{
			Balance: new(big.Int).Set(account.Balance),
			Nonce:   account.Nonce,
			Code:    append([]byte(nil), account.Code...),
			Storage: copyStorage(account.Storage),
		}
	}

	return result
}

// copyStorage creates a deep copy of a storage map
func copyStorage(storage map[common.Hash]common.Hash) map[common.Hash]common.Hash {
	if storage == nil {
		return nil
	}
	result := make(map[common.Hash]common.Hash, len(storage))
	for key, value := range storage {
		result[key] = value
	}
	return result
}
