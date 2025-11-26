package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa/utils"
)

// PredeterminedAddresses represents the mapping from contract names to their predetermined deployment addresses
// along with an optional deployment order.
type PredeterminedAddresses struct {
	// DeploymentOrder specifies the order in which contracts should be deployed.
	// If empty, no specific order is enforced.
	DeploymentOrder []string `json:"deployment_order"`

	// LibraryAddresses maps contract/library names to their predetermined deployment addresses.
	LibraryAddresses map[string]common.Address `json:"library_addresses"`
}

// combinedSolcLinkFormat represents the JSON structure used by crytic-export/combined_solc.link
type combinedSolcLinkFormat struct {
	DeploymentOrder  []string          `json:"deployment_order"`
	LibraryAddresses map[string]string `json:"library_addresses"`
}

// ReadPredeterminedAddresses reads the predetermined addresses from a JSON file.
// The expected format is crytic-export/combined_solc.link format:
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
// Returns a PredeterminedAddresses struct or an error if reading or parsing fails.
func ReadPredeterminedAddresses(filePath string) (*PredeterminedAddresses, error) {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read predetermined addresses file %s: %w", filePath, err)
	}

	// Parse the JSON into the combined_solc.link format
	var linkFormat combinedSolcLinkFormat
	err = json.Unmarshal(data, &linkFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to parse predetermined addresses JSON from %s: %w", filePath, err)
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

	return &PredeterminedAddresses{
		DeploymentOrder:  linkFormat.DeploymentOrder,
		LibraryAddresses: addresses,
	}, nil
}
