package contracts

import (
	"strings"

	"github.com/crytic/medusa/compilation/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// Contracts describes an array of contracts
type Contracts []*Contract

// MatchBytecode takes init and/or runtime bytecode and attempts to match it to a contract definition in the
// current list of contracts. It returns the contract definition if found. Otherwise, it returns nil.
func (c Contracts) MatchBytecode(initBytecode []byte, runtimeBytecode []byte) *Contract {
	// Loop through all our contract definitions to find a match.
	for i := 0; i < len(c); i++ {
		// If we have a match, register the deployed contract.
		if c[i].CompiledContract().IsMatch(initBytecode, runtimeBytecode) {
			return c[i]
		}
	}

	// If we found no definition, return nil.
	return nil
}

// Contract describes a compiled smart contract.
type Contract struct {
	// name represents the name of the contract.
	name string

	// sourcePath represents the key used to index the source file in the compilation it was derived from.
	sourcePath string

	// compiledContract describes the compiled contract data.
	compiledContract *types.CompiledContract

	// compilation describes the compilation which contains the compiledContract.
	compilation *types.Compilation

	// candidateMethods are the methods that can be called on the contract after targeting/excluding is performed.
	candidateMethods []abi.Method

	// propertyTestMethods are the methods that are property tests (subset of candidateMethods).
	propertyTestMethods []abi.Method

	// optimizationTestMethods are the methods that are optimization tests (subset of candidateMethods).
	optimizationTestMethods []abi.Method

	// assertionTestMethods are the methods that are assertion tests (subset of candidateMethods).
	// All candidateMethods that are not property or optimization tests are assertion tests.
	assertionTestMethods []abi.Method
}

// NewContract returns a new Contract instance with the provided information.
func NewContract(name string, sourcePath string, compiledContract *types.CompiledContract, compilation *types.Compilation) *Contract {
	var methods []abi.Method
	for _, method := range compiledContract.Abi.Methods {
		methods = append(methods, method)
	}
	return &Contract{
		name:             name,
		sourcePath:       sourcePath,
		compiledContract: compiledContract,
		compilation:      compilation,
		candidateMethods: methods,
	}
}

func containsMethod(methods []string, target string) bool {
	for _, method := range methods {
		if method == target {
			return true
		}
	}
	return false
}

func (c *Contract) WithTargetMethods(target []string) *Contract {
	var candidateMethods []abi.Method
	for _, method := range c.candidateMethods {
		canonicalSig := strings.Join([]string{c.name, method.Sig}, ".")
		if containsMethod(target, canonicalSig) {
			candidateMethods = append(candidateMethods, method)
		}
	}
	c.candidateMethods = candidateMethods
	return c
}

func (c *Contract) WithExcludedMethods(excludedMethods []string) *Contract {
	var candidateMethods []abi.Method
	for _, method := range c.candidateMethods {
		canonicalSig := strings.Join([]string{c.name, method.Sig}, ".")
		if !containsMethod(excludedMethods, canonicalSig) {
			candidateMethods = append(candidateMethods, method)
		}
	}
	c.candidateMethods = candidateMethods
	return c
}

// Name returns the name of the contract.
func (c *Contract) Name() string {
	return c.name
}

// SourcePath returns the path of the source file containing the contract.
func (c *Contract) SourcePath() string {
	return c.sourcePath
}

// CompiledContract returns the compiled contract information including source mappings, byte code, and ABI.
func (c *Contract) CompiledContract() *types.CompiledContract {
	return c.compiledContract
}

// Compilation returns the compilation which contains the CompiledContract.
func (c *Contract) Compilation() *types.Compilation {
	return c.compilation
}

// CandidateMethods returns the methods that can be called on the contract after targeting/excluding is performed.
func (c *Contract) CandidateMethods() []abi.Method {
	return c.candidateMethods
}

// PropertyTestMethods returns the methods that are property tests (subset of CandidateMethods).
func (c *Contract) PropertyTestMethods() []abi.Method {
	return c.propertyTestMethods
}

// OptimizationTestMethods returns the methods that are optimization tests (subset of CandidateMethods).
func (c *Contract) OptimizationTestMethods() []abi.Method {
	return c.optimizationTestMethods
}

// AssertionTestMethods returns the methods that are assertion tests (subset of CandidateMethods).
// All CandidateMethods that are not property or optimization tests are assertion tests.
func (c *Contract) AssertionTestMethods() []abi.Method {
	return c.assertionTestMethods
}

func (c *Contract) AddTestMethods(assertion, property, optimization []abi.Method) {
	c.assertionTestMethods = assertion
	c.propertyTestMethods = property
	c.optimizationTestMethods = optimization
}
