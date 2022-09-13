package utils

import (
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"strings"
)

// HexStringToAddress converts a hex string (with or without the "0x" prefix) to a common.Address. Returns the parsed
// address, or an error if one occurs during conversion.
func HexStringToAddress(addressHexString string) (common.Address, error) {
	// Remove the 0x prefix and decode the hex string into a byte array
	trimmedString := strings.TrimPrefix(addressHexString, "0x")

	// Pad the hex string with a 0 if its odd-length.
	if len(trimmedString)%2 != 0 {
		trimmedString = "0" + trimmedString
	}

	// Decode the hex string into a byte array
	b, err := hex.DecodeString(trimmedString)
	if err != nil {
		return common.Address{}, err
	}

	// Parse the bytes as an address and return them.
	address := common.Address{}
	address.SetBytes(b)
	return address, nil
}

// HexStringsToAddresses converts hex strings (with or without the "0x" prefix) to common.Address objects. Returns the
// parsed address, or an error if one occurs during conversion.
func HexStringsToAddresses(addressHexStrings []string) ([]common.Address, error) {
	// Create our array of address types
	addresses := make([]common.Address, 0)

	// Convert all hex strings to address types
	for _, addressHexString := range addressHexStrings {
		address, err := HexStringToAddress(addressHexString)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, address)
	}
	return addresses, nil
}
