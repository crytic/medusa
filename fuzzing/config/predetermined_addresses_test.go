package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPredeterminedAddresses(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	t.Run("ValidCombinedSolcLinkFormat", func(t *testing.T) {
		// Create a valid combined_solc.link format file
		addressesFile := filepath.Join(tempDir, "valid_combined_solc.link")
		content := `{
  "deployment_order": ["Library1", "Library2", "Library3"],
  "library_addresses": {
    "Library1": "0x000000000000000000000000000000000000a070",
    "Library2": "0x000000000000000000000000000000000000a071",
    "Library3": "0x000000000000000000000000000000000000a072"
  }
}`
		err := os.WriteFile(addressesFile, []byte(content), 0644)
		require.NoError(t, err)

		// Read the addresses
		addresses, err := ReadPredeterminedAddresses(addressesFile)
		require.NoError(t, err)
		assert.NotNil(t, addresses)
		assert.Len(t, addresses.LibraryAddresses, 3)
		assert.Len(t, addresses.DeploymentOrder, 3)

		// Verify deployment order
		assert.Equal(t, []string{"Library1", "Library2", "Library3"}, addresses.DeploymentOrder)

		// Verify addresses
		expectedAddr1 := common.HexToAddress("0x000000000000000000000000000000000000a070")
		expectedAddr2 := common.HexToAddress("0x000000000000000000000000000000000000a071")
		expectedAddr3 := common.HexToAddress("0x000000000000000000000000000000000000a072")
		assert.Equal(t, expectedAddr1, addresses.LibraryAddresses["Library1"])
		assert.Equal(t, expectedAddr2, addresses.LibraryAddresses["Library2"])
		assert.Equal(t, expectedAddr3, addresses.LibraryAddresses["Library3"])
	})

	t.Run("ValidWithoutDeploymentOrder", func(t *testing.T) {
		// Create a file without deployment order
		addressesFile := filepath.Join(tempDir, "no_order.json")
		content := `{
  "deployment_order": [],
  "library_addresses": {
    "ContractA": "0x1234567890123456789012345678901234567890",
    "ContractB": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
  }
}`
		err := os.WriteFile(addressesFile, []byte(content), 0644)
		require.NoError(t, err)

		// Read the addresses
		addresses, err := ReadPredeterminedAddresses(addressesFile)
		require.NoError(t, err)
		assert.NotNil(t, addresses)
		assert.Len(t, addresses.LibraryAddresses, 2)
		assert.Empty(t, addresses.DeploymentOrder)
	})

	t.Run("InvalidAddress", func(t *testing.T) {
		// Create an invalid addresses file
		addressesFile := filepath.Join(tempDir, "invalid_addresses.json")
		content := `{
  "deployment_order": [],
  "library_addresses": {
    "ContractA": "not-a-valid-address"
  }
}`
		err := os.WriteFile(addressesFile, []byte(content), 0644)
		require.NoError(t, err)

		// Read the addresses - should fail
		_, err = ReadPredeterminedAddresses(addressesFile)
		assert.Error(t, err)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		// Create a malformed JSON file
		addressesFile := filepath.Join(tempDir, "malformed.json")
		content := `{invalid json`
		err := os.WriteFile(addressesFile, []byte(content), 0644)
		require.NoError(t, err)

		// Read the addresses - should fail
		_, err = ReadPredeterminedAddresses(addressesFile)
		assert.Error(t, err)
	})

	t.Run("FileNotExists", func(t *testing.T) {
		// Try to read a non-existent file
		_, err := ReadPredeterminedAddresses(filepath.Join(tempDir, "nonexistent.json"))
		assert.Error(t, err)
	})

	t.Run("EmptyLibraryAddresses", func(t *testing.T) {
		// Create a file with empty library addresses
		addressesFile := filepath.Join(tempDir, "empty.json")
		content := `{
  "deployment_order": [],
  "library_addresses": {}
}`
		err := os.WriteFile(addressesFile, []byte(content), 0644)
		require.NoError(t, err)

		// Read the addresses - should succeed with empty map
		addresses, err := ReadPredeterminedAddresses(addressesFile)
		require.NoError(t, err)
		assert.NotNil(t, addresses)
		assert.Empty(t, addresses.LibraryAddresses)
	})

	t.Run("AddressWithoutPrefix", func(t *testing.T) {
		// Create an addresses file with addresses without 0x prefix
		addressesFile := filepath.Join(tempDir, "no_prefix.json")
		content := `{
  "deployment_order": [],
  "library_addresses": {
    "ContractA": "1234567890123456789012345678901234567890"
  }
}`
		err := os.WriteFile(addressesFile, []byte(content), 0644)
		require.NoError(t, err)

		// Read the addresses - utils.HexStringToAddress should handle this
		addresses, err := ReadPredeterminedAddresses(addressesFile)
		require.NoError(t, err)
		assert.Len(t, addresses.LibraryAddresses, 1)
	})
}
