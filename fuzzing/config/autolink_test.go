package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/crytic/medusa-geth/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadAutolinkConfig(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	t.Run("ValidCombinedSolcLinkFormat", func(t *testing.T) {
		// Create a valid combined_solc.link format file
		configFile := filepath.Join(tempDir, "valid_combined_solc.link")
		content := `{
  "deployment_order": ["Library1", "Library2", "Library3"],
  "library_addresses": {
    "Library1": "0x000000000000000000000000000000000000a070",
    "Library2": "0x000000000000000000000000000000000000a071",
    "Library3": "0x000000000000000000000000000000000000a072"
  }
}`
		err := os.WriteFile(configFile, []byte(content), 0644)
		require.NoError(t, err)

		// Read the config
		config, err := ReadAutolinkConfig(configFile)
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Len(t, config.LibraryAddresses, 3)
		assert.Len(t, config.DeploymentOrder, 3)

		// Verify deployment order
		assert.Equal(t, []string{"Library1", "Library2", "Library3"}, config.DeploymentOrder)

		// Verify addresses
		expectedAddr1 := common.HexToAddress("0x000000000000000000000000000000000000a070")
		expectedAddr2 := common.HexToAddress("0x000000000000000000000000000000000000a071")
		expectedAddr3 := common.HexToAddress("0x000000000000000000000000000000000000a072")
		assert.Equal(t, expectedAddr1, config.LibraryAddresses["Library1"])
		assert.Equal(t, expectedAddr2, config.LibraryAddresses["Library2"])
		assert.Equal(t, expectedAddr3, config.LibraryAddresses["Library3"])
	})

	t.Run("ValidWithoutDeploymentOrder", func(t *testing.T) {
		// Create a file without deployment order
		configFile := filepath.Join(tempDir, "no_order.json")
		content := `{
  "deployment_order": [],
  "library_addresses": {
    "ContractA": "0x1234567890123456789012345678901234567890",
    "ContractB": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
  }
}`
		err := os.WriteFile(configFile, []byte(content), 0644)
		require.NoError(t, err)

		// Read the config
		config, err := ReadAutolinkConfig(configFile)
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Len(t, config.LibraryAddresses, 2)
		assert.Empty(t, config.DeploymentOrder)
	})

	t.Run("InvalidAddress", func(t *testing.T) {
		// Create an invalid addresses file
		configFile := filepath.Join(tempDir, "invalid_addresses.json")
		content := `{
  "deployment_order": [],
  "library_addresses": {
    "ContractA": "not-a-valid-address"
  }
}`
		err := os.WriteFile(configFile, []byte(content), 0644)
		require.NoError(t, err)

		// Read the config - should fail
		_, err = ReadAutolinkConfig(configFile)
		assert.Error(t, err)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		// Create a malformed JSON file
		configFile := filepath.Join(tempDir, "malformed.json")
		content := `{invalid json`
		err := os.WriteFile(configFile, []byte(content), 0644)
		require.NoError(t, err)

		// Read the config - should fail
		_, err = ReadAutolinkConfig(configFile)
		assert.Error(t, err)
	})

	t.Run("FileNotExists", func(t *testing.T) {
		// Try to read a non-existent file
		_, err := ReadAutolinkConfig(filepath.Join(tempDir, "nonexistent.json"))
		assert.Error(t, err)
	})

	t.Run("EmptyLibraryAddresses", func(t *testing.T) {
		// Create a file with empty library addresses
		configFile := filepath.Join(tempDir, "empty.json")
		content := `{
  "deployment_order": [],
  "library_addresses": {}
}`
		err := os.WriteFile(configFile, []byte(content), 0644)
		require.NoError(t, err)

		// Read the config - should succeed with empty map
		config, err := ReadAutolinkConfig(configFile)
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Empty(t, config.LibraryAddresses)
	})

	t.Run("AddressWithoutPrefix", func(t *testing.T) {
		// Create a config file with addresses without 0x prefix
		configFile := filepath.Join(tempDir, "no_prefix.json")
		content := `{
  "deployment_order": [],
  "library_addresses": {
    "ContractA": "1234567890123456789012345678901234567890"
  }
}`
		err := os.WriteFile(configFile, []byte(content), 0644)
		require.NoError(t, err)

		// Read the config - utils.HexStringToAddress should handle this
		config, err := ReadAutolinkConfig(configFile)
		require.NoError(t, err)
		assert.Len(t, config.LibraryAddresses, 1)
	})
}
