package utils

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/params"
)

// CopyChainConfig takes a chain configuration and creates a copy.
// Returns the copy of the chain configuration, or an error if one occurs.
func CopyChainConfig(config *params.ChainConfig) (*params.ChainConfig, error) {
	// TODO: This is not performant. It should be replaced with something more performant in the future.

	// Encode the chain config.
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	// Decode a new chain config from the encoded data.
	var chainConfig *params.ChainConfig
	err = json.Unmarshal(data, &chainConfig)
	if err != nil {
		return nil, err
	}

	// Return it.
	return chainConfig, nil
}
