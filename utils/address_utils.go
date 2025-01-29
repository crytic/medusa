package utils

import (
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
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

// ResolveAddressToLabelFromElements parse array elements for ethereum addresses and if found replace that with a label if one exists
func ResolveAddressToLabelFromElements(elements []any, addressToLabel map[common.Address]string) []any {
	updatedElements := []any{}
	for _, element := range elements {
		// Check if the element is a string
		if str, ok := element.(string); ok {
			// Replace addresses in the string with their labels
			updatedElements = append(updatedElements, ResolveAddressToLabelFromString(str, addressToLabel))
		} else {
			// Keep non-string elements unchanged
			updatedElements = append(updatedElements, element)
		}
	}

	return updatedElements
}

// ResolveAddressToLabelFromString parse a string for ethereum addresses and if found replace that with a label if one exists
func ResolveAddressToLabelFromString(str string, addressToLabel map[common.Address]string) string {
	addressRegex := regexp.MustCompile(`0x[a-fA-F0-9]{40}`)
	processedString := addressRegex.ReplaceAllStringFunc(str, func(match string) string {
		address := common.HexToAddress(match) // Convert the match to an Ethereum address
		if label, exists := addressToLabel[address]; exists {
			//fmt.Printf("Replacing address %s with label: %s\n", match, label)
			return label // Replace address with label
		}
		trimmedAddress := TrimLeadingZeroesFromAddress(match)
		//fmt.Printf("Address %s does not have a label, trimming leading zeroes to: %s\n", match, trimmedAddress)
		return trimmedAddress // Keep the address unchanged if no label is found
	})

	return processedString
}

// TrimLeadingZeroesFromAddress removes the leading zeroes from an address for readability
// Example: sender=0x0000000000000000000000000000000000030000 becomes sender=0x30000 when shown on console
func TrimLeadingZeroesFromAddress(hexString string) string {
	if strings.HasPrefix(hexString, "0x") {
		// Retain "0x" and trim leading zeroes from the rest of the string
		return "0x" + strings.TrimLeft(hexString[2:], "0")
	}
	// Trim leading zeroes if there's no "0x" prefix
	return strings.TrimLeft(hexString, "0")
}
