package valuegeneration

import (
	"math/big"
	"strings"

	compilationTypes "github.com/crytic/medusa/compilation/types"
	"github.com/ethereum/go-ethereum/common"
)

// SeedFromAst allows a ValueSet to be seeded from an AST interface.
func (vs *ValueSet) SeedFromSlither(constants []compilationTypes.ConstantUsed) {
	for _, constant := range constants {

		if strings.HasPrefix(constant.Type, "uint") || strings.HasPrefix(constant.Type, "int") {
			var constantBigInt, _ = new(big.Int).SetString(constant.Value, 10)
			vs.AddInteger(constantBigInt)
		} else if constant.Type == "bool" {
			if constant.Value == "False" {
				vs.AddInteger(big.NewInt(0))
			} else {
				vs.AddInteger(big.NewInt(1))
			}
		} else if constant.Type == "string" {
			vs.AddString(constant.Value)
		} else if constant.Type == "address" {
			var addressBigInt, _ = new(big.Int).SetString(constant.Value, 10)
			vs.AddAddress(common.BigToAddress(addressBigInt))
		}

	}
}
