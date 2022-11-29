package valuegeneration

import (
	"github.com/ethereum/go-ethereum/common"
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

			// Seed ValueSet with literals
			if literalKind == "number" {
				if strings.HasPrefix(literalValue, "0x") {
					if b, ok := big.NewInt(0).SetString(literalValue[2:], 16); ok {
						vs.AddInteger(b)
						vs.AddInteger(new(big.Int).Neg(b))
						vs.AddAddress(common.BigToAddress(b))
					}
				} else {
					if b, ok := big.NewInt(0).SetString(literalValue, 10); ok {
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
