package types

import (
	"encoding/json"
	"strings"
)

// ExtractedCall represents a function call extracted from AST analysis
type ExtractedCall struct {
	ContractName string
	FunctionName string
}

// IsTestContract determines if a contract is a test contract based on naming patterns
func IsTestContract(contract ContractDefinition) bool {
	nameLower := strings.ToLower(contract.CanonicalName)
	return strings.Contains(nameLower, "test")
}

// IsTestFunction determines if a function is a test function based on naming patterns
func IsTestFunction(function FunctionDefinition) bool {
	if function.Name == "" {
		return false
	}
	nameLower := strings.ToLower(function.Name)
	return strings.HasPrefix(nameLower, "test") ||
		strings.HasPrefix(nameLower, "invariant_") ||
		strings.HasPrefix(nameLower, "testfuzz")
}

// ExtractFunctionCalls extracts all external function calls from a function definition
func ExtractFunctionCalls(function FunctionDefinition) ([]ExtractedCall, error) {
	if function.Body == nil {
		return nil, nil
	}

	calls := make([]ExtractedCall, 0)

	// Process each statement in the function body
	for _, stmtData := range function.Body.Statements {
		extractedCalls := extractCallsFromStatement(stmtData)
		calls = append(calls, extractedCalls...)
	}

	return calls, nil
}

// extractCallsFromStatement recursively extracts calls from a statement
func extractCallsFromStatement(stmtData json.RawMessage) []ExtractedCall {
	calls := make([]ExtractedCall, 0)

	// Parse the node type
	var nodeType struct {
		NodeType string `json:"nodeType"`
	}
	if err := json.Unmarshal(stmtData, &nodeType); err != nil {
		return calls
	}

	switch nodeType.NodeType {
	case "ExpressionStatement":
		var exprStmt ExpressionStatement
		if err := json.Unmarshal(stmtData, &exprStmt); err != nil {
			return calls
		}
		calls = append(calls, extractCallsFromExpression(exprStmt.Expression)...)

	case "Block":
		var block Block
		if err := json.Unmarshal(stmtData, &block); err != nil {
			return calls
		}
		for _, innerStmt := range block.Statements {
			calls = append(calls, extractCallsFromStatement(innerStmt)...)
		}

	case "IfStatement":
		var ifStmt struct {
			TrueBody  json.RawMessage `json:"trueBody"`
			FalseBody json.RawMessage `json:"falseBody"`
		}
		if err := json.Unmarshal(stmtData, &ifStmt); err != nil {
			return calls
		}
		if ifStmt.TrueBody != nil {
			calls = append(calls, extractCallsFromStatement(ifStmt.TrueBody)...)
		}
		if ifStmt.FalseBody != nil {
			calls = append(calls, extractCallsFromStatement(ifStmt.FalseBody)...)
		}

	case "ForStatement", "WhileStatement":
		var loopStmt struct {
			Body json.RawMessage `json:"body"`
		}
		if err := json.Unmarshal(stmtData, &loopStmt); err != nil {
			return calls
		}
		if loopStmt.Body != nil {
			calls = append(calls, extractCallsFromStatement(loopStmt.Body)...)
		}

	case "VariableDeclarationStatement":
		var varDecl struct {
			InitialValue json.RawMessage `json:"initialValue"`
		}
		if err := json.Unmarshal(stmtData, &varDecl); err != nil {
			return calls
		}
		if varDecl.InitialValue != nil {
			calls = append(calls, extractCallsFromExpression(varDecl.InitialValue)...)
		}
	}

	return calls
}

// extractCallsFromExpression extracts calls from an expression node
func extractCallsFromExpression(exprData json.RawMessage) []ExtractedCall {
	calls := make([]ExtractedCall, 0)

	// Parse the node type
	var nodeType struct {
		NodeType string `json:"nodeType"`
	}
	if err := json.Unmarshal(exprData, &nodeType); err != nil {
		return calls
	}

	switch nodeType.NodeType {
	case "FunctionCall":
		var funcCall FunctionCall
		if err := json.Unmarshal(exprData, &funcCall); err != nil {
			return calls
		}

		// Extract the call information
		if callInfo := extractCallInfo(funcCall); callInfo != nil {
			calls = append(calls, *callInfo)
		}

		// Also check arguments for nested calls
		for _, arg := range funcCall.Arguments {
			calls = append(calls, extractCallsFromExpression(arg)...)
		}

	case "Assignment":
		var assignment struct {
			LeftHandSide  json.RawMessage `json:"leftHandSide"`
			RightHandSide json.RawMessage `json:"rightHandSide"`
		}
		if err := json.Unmarshal(exprData, &assignment); err != nil {
			return calls
		}
		if assignment.RightHandSide != nil {
			calls = append(calls, extractCallsFromExpression(assignment.RightHandSide)...)
		}

	case "BinaryOperation":
		var binOp struct {
			LeftExpression  json.RawMessage `json:"leftExpression"`
			RightExpression json.RawMessage `json:"rightExpression"`
		}
		if err := json.Unmarshal(exprData, &binOp); err != nil {
			return calls
		}
		if binOp.LeftExpression != nil {
			calls = append(calls, extractCallsFromExpression(binOp.LeftExpression)...)
		}
		if binOp.RightExpression != nil {
			calls = append(calls, extractCallsFromExpression(binOp.RightExpression)...)
		}
	}

	return calls
}

// extractCallInfo extracts contract and function names from a FunctionCall node
func extractCallInfo(funcCall FunctionCall) *ExtractedCall {
	// Parse the expression to determine if it's a member access (contract.function)
	var exprType struct {
		NodeType string `json:"nodeType"`
	}
	if err := json.Unmarshal(funcCall.Expression, &exprType); err != nil {
		return nil
	}

	if exprType.NodeType == "MemberAccess" {
		var memberAccess MemberAccess
		if err := json.Unmarshal(funcCall.Expression, &memberAccess); err != nil {
			return nil
		}

		// Extract the contract name from the member access expression
		contractName := extractContractName(memberAccess.Expression)
		functionName := memberAccess.MemberName

		// Filter out built-in functions
		if isBuiltinFunction(functionName) {
			return nil
		}

		if contractName != "" && functionName != "" {
			return &ExtractedCall{
				ContractName: contractName,
				FunctionName: functionName,
			}
		}
	}

	return nil
}

// extractContractName extracts the contract name from an expression
func extractContractName(exprData json.RawMessage) string {
	// Parse the node type
	var nodeType struct {
		NodeType string `json:"nodeType"`
	}
	if err := json.Unmarshal(exprData, &nodeType); err != nil {
		return ""
	}

	if nodeType.NodeType == "Identifier" {
		var identifier Identifier
		if err := json.Unmarshal(exprData, &identifier); err != nil {
			return ""
		}
		return identifier.Name
	}

	// Could also be a member access (e.g., contracts.myContract)
	// For simplicity, we only handle direct identifiers
	return ""
}

// isBuiltinFunction checks if a function name is a Solidity built-in
func isBuiltinFunction(name string) bool {
	builtins := map[string]bool{
		"require":      true,
		"assert":       true,
		"revert":       true,
		"selfdestruct": true,
		"keccak256":    true,
		"sha256":       true,
		"ripemd160":    true,
		"ecrecover":    true,
		"addmod":       true,
		"mulmod":       true,
	}
	return builtins[name]
}
