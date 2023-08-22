package valuegeneration

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"math/big"
	"strings"
)

// SeedFromAst allows a ValueSet to be seeded from an AST interface.
func (vs *ValueSet) SeedFromAst(ast any) {
	// Walk our AST while extracting values
	walkAstNodes(ast, func(node map[string]any) {
		// Extract values depending on node type.
		nodeType, obtainedNodeType := node["nodeType"].(string)
		if obtainedNodeType && strings.EqualFold(nodeType, "Literal") {
			// Extract literal kind and value
			literalKind, obtainedKind := node["kind"].(string)
			literalValue, obtainedValue := node["value"].(string)
			if !obtainedKind || !obtainedValue {
				return // fail silently to continue walking
			}

			// Extract the subdenomination type
			tempSubdenomination, obtainedSubdenomination := node["subdenomination"].(string)
			var literalSubdenomination *string
			if obtainedSubdenomination {
				literalSubdenomination = &tempSubdenomination
			}

			// Seed ValueSet with literals
			if literalKind == "number" {
				// If it has a 0x prefix, it won't have decimals
				if strings.HasPrefix(literalValue, "0x") {
					if b, ok := big.NewInt(0).SetString(literalValue[2:], 16); ok {
						vs.AddInteger(b)
						vs.AddInteger(new(big.Int).Neg(b))
						vs.AddAddress(common.BigToAddress(b))
					}
				} else {
					if decValue, err := decimal.NewFromString(literalValue); err == nil {
						b := getAbsoluteValueFromDenominatedValue(decValue, literalSubdenomination)
						vs.AddInteger(b)
						vs.AddInteger(new(big.Int).Neg(b))
						vs.AddAddress(common.BigToAddress(b))
					}
				}
			} else if literalKind == "string" {
				vs.AddString(literalValue)
			}
		}
	})
}

// getAbsoluteValueFromDenominatedValue converts a given decimal number in a provided denomination to a big.Int
// that represents its actual calculated value.
// Note: Decimals must be used as big.Float is prone to similar mantissa-related precision issues as float32/float64.
// Returns the calculated value given the floating point number in a given denomination.
func getAbsoluteValueFromDenominatedValue(number decimal.Decimal, denomination *string) *big.Int {
	// If the denomination is nil, we do nothing
	if denomination == nil {
		return number.BigInt()
	}

	// Otherwise, switch on the type and obtain a multiplier
	var multiplier decimal.Decimal
	switch *denomination {
	case "wei":
		multiplier = decimal.NewFromFloat32(1)
	case "gwei":
		multiplier = decimal.NewFromFloat32(1e9)
	case "szabo":
		multiplier = decimal.NewFromFloat32(1e12)
	case "finney":
		multiplier = decimal.NewFromFloat32(1e15)
	case "ether":
		multiplier = decimal.NewFromFloat32(1e18)
	case "seconds":
		multiplier = decimal.NewFromFloat32(1)
	case "minutes":
		multiplier = decimal.NewFromFloat32(60)
	case "hours":
		multiplier = decimal.NewFromFloat32(60 * 60)
	case "days":
		multiplier = decimal.NewFromFloat32(60 * 60 * 24)
	case "weeks":
		multiplier = decimal.NewFromFloat32(60 * 60 * 24 * 7)
	case "years":
		multiplier = decimal.NewFromFloat32(60 * 60 * 24 * 7 * 365)
	default:
		multiplier = decimal.NewFromFloat32(1)
	}

	// Obtain the transformed number as an integer.
	transformedValue := number.Mul(multiplier)
	return transformedValue.BigInt()
}

// walkAstNodes walks/iterates across an AST for each node, calling the provided walk function with each discovered node
// as an argument.
func walkAstNodes(ast any, walkFunc func(node map[string]any)) {
	// Try to parse our node as different types and walk all children.
	if d, ok := ast.(map[string]any); ok {
		// If this dictionary contains keys 'id' and 'nodeType', we can assume it's an AST node
		_, hasId := d["id"]
		_, hasNodeType := d["nodeType"]
		if hasId && hasNodeType {
			walkFunc(d)
		}

		// Walk all keys of the dictionary.
		for _, v := range d {
			walkAstNodes(v, walkFunc)
		}
	} else if slice, ok := ast.([]any); ok {
		// Walk all elements of a slice.
		for _, elem := range slice {
			walkAstNodes(elem, walkFunc)
		}
	}
}
