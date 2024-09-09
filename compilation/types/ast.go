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
	// GetNodeType returns solc's node type e.g. FunctionDefinition, ContractDefinition.
	GetNodeType() string
}

// FunctionDefinition is the function definition node
type FunctionDefinition struct {
	// NodeType represents the node type (currently we only evaluate source unit node types)
	NodeType string `json:"nodeType"`
	// Src is the source file for this AST
	Src  string `json:"src"`
	Name string `json:"name,omitempty"`
}

func (s FunctionDefinition) GetNodeType() string {
	return s.NodeType
}

// ContractDefinition is the contract definition node
type ContractDefinition struct {
	// NodeType represents the node type (currently we only evaluate source unit node types)
	NodeType string `json:"nodeType"`
	// Nodes is a list of Nodes within the AST
	Nodes []Node `json:"nodes"`
	// Src is the source file for this AST
	Src string `json:"src"`
	// CanonicalName is the name of the contract definition
	CanonicalName string `json:"canonicalName,omitempty"`
	// Kind is a ContractKind that represents what type of contract definition this is (contract, interface, or library)
	Kind ContractKind `json:"contractKind,omitempty"`
}

func (s ContractDefinition) GetNodeType() string {
	return s.NodeType
}

func (c *ContractDefinition) UnmarshalJSON(data []byte) error {
	// Unmarshal the top-level AST into our own representation. Defer the unmarshaling of all the individual nodes until later
	type Alias ContractDefinition
	aux := &struct {
		Nodes []json.RawMessage `json:"nodes"`

		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Iterate through all the nodes of the contract definition
	for _, nodeData := range aux.Nodes {
		// Unmarshal the node data to retrieve the node type
		var nodeType struct {
			NodeType string `json:"nodeType"`
		}
		if err := json.Unmarshal(nodeData, &nodeType); err != nil {
			return err
		}

		// Unmarshal the contents of the node based on the node type
		switch nodeType.NodeType {
		case "FunctionDefinition":
			// If this is a function definition, unmarshal it
			var functionDefinition FunctionDefinition
			if err := json.Unmarshal(nodeData, &functionDefinition); err != nil {
				return err
			}
			c.Nodes = append(c.Nodes, functionDefinition)
		default:
			continue
		}
	}

	return nil

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
		switch nodeType.NodeType {
		case "ContractDefinition":
			// If this is a contract definition, unmarshal it
			var contractDefinition ContractDefinition
			if err := json.Unmarshal(nodeData, &contractDefinition); err != nil {
				return err
			}
			a.Nodes = append(a.Nodes, contractDefinition)

		case "FunctionDefinition":
			// If this is a function definition, unmarshal it
			var functionDefinition FunctionDefinition
			if err := json.Unmarshal(nodeData, &functionDefinition); err != nil {
				return err
			}
			a.Nodes = append(a.Nodes, functionDefinition)

		// TODO: Add cases for other node types as needed
		default:
			continue
		}

	}

	return nil
}

// GetSrcMapSourceUnitID returns the source unit ID based on the source of the AST
func GetSrcMapSourceUnitID(src string) int {
	re := regexp.MustCompile(`[0-9]*:[0-9]*:([0-9]*)`)
	sourceUnitCandidates := re.FindStringSubmatch(src)

	if len(sourceUnitCandidates) == 2 { // FindStringSubmatch includes the whole match as the first element
		sourceUnit, err := strconv.Atoi(sourceUnitCandidates[1])
		if err == nil {
			return sourceUnit
		}
	}
	return -1
}

// GetSrcMapStart returns the byte offset where the function definition starts in the source file
func GetSrcMapStart(src string) int {
	// 95:42:0 returns 95
	re := regexp.MustCompile(`([0-9]*):[0-9]*:[0-9]*`)
	startCandidates := re.FindStringSubmatch(src)

	if len(startCandidates) == 2 { // FindStringSubmatch includes the whole match as the first element
		start, err := strconv.Atoi(startCandidates[1])
		if err == nil {
			return start
		}
	}
	return -1
}

// GetSrcMapLength returns the length of the function definition in bytes
func GetSrcMapLength(src string) int {
	// 95:42:0 returns 42
	re := regexp.MustCompile(`[0-9]*:([0-9]*):[0-9]*`)
	endCandidates := re.FindStringSubmatch(src)

	if len(endCandidates) == 2 { // FindStringSubmatch includes the whole match as the first element
		end, err := strconv.Atoi(endCandidates[1])
		if err == nil {
			return end
		}
	}
	return -1
}
