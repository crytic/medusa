package fuzzing

import (
	"math/big"
	"strings"
)

// seedFromAst allows a corpus to be seeded from an AST interface.
func (c *Corpus) seedFromAst(ast interface{}) {
	// Walk our AST while extracting values
	c.walkAstNodes(ast, func(node map[string]interface{}) {
		// Extract values depending on node type.
		nodeType, obtainedNodeType := node["nodeType"].(string)
		if obtainedNodeType && strings.EqualFold(nodeType, "Literal") {
			// Extract literal kind and value
			literalKind, obtainedKind := node["kind"].(string)
			literalValue, obtainedValue := node["value"].(string)
			if !obtainedKind || !obtainedValue {
				return // fail silently to continue walking
			}

			// Seed corpus with literals
			if literalKind == "number" {
				if b, ok := big.NewInt(0).SetString(literalValue, 10); ok {
					c.AddInteger(*b)
				}
			} else if literalKind == "string" {
				c.AddString(literalValue)
			}
		}
	})
}

// walkAstNodes walks/iterates across an AST for each node, calling the provided walk function with each discovered node
// as an argument.
func (c *Corpus) walkAstNodes(ast interface{}, walkFunc func(node map[string]interface{})) {
	// Try to parse our node as different types and walk all children.
	if d, ok := ast.(map[string]interface{}); ok {
		// If this dictionary contains keys 'id' and 'nodeType', we can assume it's an AST node
		_, hasId := d["id"]
		_, hasNodeType := d["nodeType"]
		if hasId && hasNodeType {
			walkFunc(d)
		}

		// Walk all keys of the dictionary.
		for _, v := range d {
			c.walkAstNodes(v, walkFunc)
		}
	} else if slice, ok := ast.([]interface{}); ok {
		// Walk all elements of a slice.
		for _, elem := range slice {
			c.walkAstNodes(elem, walkFunc)
		}
	}
}