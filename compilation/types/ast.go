package types

import (
	"encoding/json"
	"regexp"
	"strconv"
)

// ContractKind represents the kind of contract definition represented by an AST node
type ContractKind string

const (
	// ContractKindContract represents a contract node
	ContractKindContract ContractKind = "contract"
	// ContractKindLibrary represents a library node
	ContractKindLibrary ContractKind = "library"
	// ContractKindInterface represents an interface node
	ContractKindInterface ContractKind = "interface"
)

// Node interface represents a generic AST node
type Node interface {
	GetNodeType() string
}

// ContractDefinition is the contract definition node
type ContractDefinition struct {
	// NodeType represents the AST node type (note that it will always be a contract definition)
	NodeType string `json:"nodeType"`
	// CanonicalName is the name of the contract definition
	CanonicalName string `json:"canonicalName,omitempty"`
	// Kind is a ContractKind that represents what type of contract definition this is (contract, interface, or library)
	Kind ContractKind `json:"contractKind,omitempty"`
}

// GetNodeType implements the Node interface and returns the node type for the contract definition
func (s ContractDefinition) GetNodeType() string {
	return s.NodeType
}

// AST is the abstract syntax tree
type AST struct {
	// NodeType represents the node type (currently we only evaluate source unit node types)
	NodeType string `json:"nodeType"`
	// Nodes is a list of Nodes within the AST
	Nodes []Node `json:"nodes"`
	// Src is the source file for this AST
	Src string `json:"src"`
}

// UnmarshalJSON unmarshals from JSON
func (a *AST) UnmarshalJSON(data []byte) error {
	// Unmarshal the top-level AST into our own representation. Defer the unmarshaling of all the individual nodes until later
	type Alias AST
	aux := &struct {
		Nodes []json.RawMessage `json:"nodes"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Check if nodeType is "SourceUnit". Return early otherwise
	if aux.NodeType != "SourceUnit" {
		return nil
	}

	// Iterate through all the nodes of the source unit
	for _, nodeData := range aux.Nodes {
		// Unmarshal the node data to retrieve the node type
		var nodeType struct {
			NodeType string `json:"nodeType"`
		}
		if err := json.Unmarshal(nodeData, &nodeType); err != nil {
			return err
		}

		// Unmarshal the contents of the node based on the node type
		var node Node
		switch nodeType.NodeType {
		case "ContractDefinition":
			// If this is a contract definition, unmarshal it
			var contractDefinition ContractDefinition
			if err := json.Unmarshal(nodeData, &contractDefinition); err != nil {
				return err
			}
			node = contractDefinition
		// TODO: Add cases for other node types as needed
		default:
			continue
		}

		// Append the node
		a.Nodes = append(a.Nodes, node)
	}

	return nil
}

// GetSourceUnitID returns the source unit ID based on the source of the AST
func (a *AST) GetSourceUnitID() int {
	re := regexp.MustCompile(`[0-9]*:[0-9]*:([0-9]*)`)
	sourceUnitCandidates := re.FindStringSubmatch(a.Src)

	if len(sourceUnitCandidates) == 2 { // FindStringSubmatch includes the whole match as the first element
		sourceUnit, err := strconv.Atoi(sourceUnitCandidates[1])
		if err == nil {
			return sourceUnit
		}
	}
	return -1
}
