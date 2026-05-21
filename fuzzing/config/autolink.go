package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa/utils"
)

// AutolinkConfig represents the mapping from contract names to their autolinked deployment addresses
// along with an optional deployment order. This is read from the crytic-export/combined_solc.link file.
type AutolinkConfig struct {
	// DeploymentOrder specifies the order in which contracts should be deployed.
	// If empty, no specific order is enforced.
	DeploymentOrder []string `json:"deployment_order"`

	// LibraryAddresses maps contract/library names to their autolinked deployment addresses.
	LibraryAddresses map[string]common.Address `json:"library_addresses"`
}

// combinedSolcLinkFormat represents the JSON structure used by crytic-export/combined_solc.link
type combinedSolcLinkFormat struct {
	DeploymentOrder  []string          `json:"deployment_order"`
	LibraryAddresses map[string]string `json:"library_addresses"`
}

// ReadAutolinkConfig reads the autolink configuration from the crytic-export/combined_solc.link file.
// The expected format is:
//
//	{
//	  "deployment_order": ["Library1", "Library2", "Library3"],
//	  "library_addresses": {
//	    "Library1": "0x000000000000000000000000000000000000a070",
//	    "Library2": "0x000000000000000000000000000000000000a071",
//	    "Library3": "0x000000000000000000000000000000000000a072"
//	  }
//	}
//
// Returns an AutolinkConfig struct or an error if reading or parsing fails.
func ReadAutolinkConfig(filePath string) (*AutolinkConfig, error) {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read autolink config file %s: %w", filePath, err)
	}

	// Parse the JSON into the combined_solc.link format
	var linkFormat combinedSolcLinkFormat
	err = json.Unmarshal(data, &linkFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to parse autolink config JSON from %s: %w", filePath, err)
	}

	// Convert string addresses to common.Address
	addresses := make(map[string]common.Address)
	for contractName, addrStr := range linkFormat.LibraryAddresses {
		addr, err := utils.HexStringToAddress(addrStr)
		if err != nil {
			return nil, fmt.Errorf("invalid address %s for contract %s: %w", addrStr, contractName, err)
		}
		addresses[contractName] = addr
	}

	return &AutolinkConfig{
		DeploymentOrder:  linkFormat.DeploymentOrder,
		LibraryAddresses: addresses,
	}, nil
}
