package utils

import (
	"encoding/hex"
	"strings"

	"github.com/crytic/medusa-geth/common"
)

// HexStringToAddress converts a hex string (with or without the "0x" prefix) to a common.Address. Returns the parsed
// address, or an error if one occurs during conversion.
func HexStringToAddress(addressHexString string) (common.Address, error) {
	// Remove the 0x prefix and decode the hex string into a byte array
	trimmedString := strings.TrimPrefix(addressHexString, "0x")

	// Pad the hex string with a 0 if it's odd-length.
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

// AttachLabelToAddress appends a human-readable label to an address for console-output. If a label is not-provided,
// the address is returned back. Note that this function also trims any leading zeroes from the address to clean it
// up for console output.
// TODO: Maybe we allow the user to determine whether they want to trim the address of leading zeroes?
func AttachLabelToAddress(address common.Address, label string) string {
	trimmedHexString := TrimLeadingZeroesFromAddress(address)
	if label == "" {
		return trimmedHexString
	}
	return label + " [" + trimmedHexString + "]"
}

// TrimLeadingZeroesFromAddress removes the leading zeroes from an address for readability and returns it as a string
// Example: sender=0x0000000000000000000000000000000000030000 becomes sender=0x30000 when shown on console
func TrimLeadingZeroesFromAddress(address common.Address) string {
	hexString := address.String()
	if strings.HasPrefix(hexString, "0x") {
		// Retain "0x" and trim leading zeroes from the rest of the string
		trimmed := strings.TrimLeft(hexString[2:], "0")
		if trimmed == "" {
			return "0x0"
		}
		return "0x" + trimmed
	}
	// Trim leading zeroes if there's no "0x" prefix
	trimmed := strings.TrimLeft(hexString, "0")
	if trimmed == "" {
		return "0"
	}
	return trimmed
}
