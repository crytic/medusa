package valuegeneration

import (
	"math/big"
	"strings"

	compilationTypes "github.com/crytic/medusa/compilation/types"
	"github.com/ethereum/go-ethereum/common"
)

// SeedFromSlither allows a ValueSet to be seeded from the output of slither.
func (vs *ValueSet) SeedFromSlither(slither *compilationTypes.SlitherResults) {
	// Iterate across all the constants
	for _, constant := range slither.Constants {
		// Capture uint/int types
		if strings.HasPrefix(constant.Type, "uint") || strings.HasPrefix(constant.Type, "int") {
			var b, _ = new(big.Int).SetString(constant.Value, 10)
			vs.AddInteger(b)
			vs.AddInteger(new(big.Int).Neg(b))
			vs.AddBytes(b.Bytes())
		} else if constant.Type == "bool" {
			// Capture booleans
			if constant.Value == "False" {
				vs.AddInteger(big.NewInt(0))
			} else {
				vs.AddInteger(big.NewInt(1))
			}
		} else if constant.Type == "string" {
			// Capture strings
			vs.AddString(constant.Value)
			vs.AddBytes([]byte(constant.Value))
		} else if constant.Type == "address" {
			// Capture addresses
			var addressBigInt, _ = new(big.Int).SetString(constant.Value, 10)
			vs.AddAddress(common.BigToAddress(addressBigInt))
			vs.AddBytes([]byte(constant.Value))
		}
	}
}
