package utils

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"strings"
)

// HexStringToAddress converts a hex string (with or without the "0x" prefix) to a common.Address. Returns the parsed
// address, or an error if one occurs during conversion.
func HexStringToAddress(s string) (*common.Address, error) {
	// Remove the 0x prefix and decode the hex string into a byte array
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return nil, err
	}

	// Parse the bytes as an address and return them.
	address := common.Address{}
	address.SetBytes(b)
	return &address, nil
}
