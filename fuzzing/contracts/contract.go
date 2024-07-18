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

	// propertyTestMethods are the methods that are property tests.
	propertyTestMethods []abi.Method

	// optimizationTestMethods are the methods that are optimization tests.
	optimizationTestMethods []abi.Method

	// assertionTestMethods are ALL other methods that are not property or optimization tests by default.
	// If configured, the methods will be targeted or excluded based on the targetFunctionSignatures
	// and excludedFunctionSignatures, respectively.
	assertionTestMethods []abi.Method
}

// NewContract returns a new Contract instance with the provided information.
func NewContract(name string, sourcePath string, compiledContract *types.CompiledContract, compilation *types.Compilation) *Contract {
	return &Contract{
		name:             name,
		sourcePath:       sourcePath,
		compiledContract: compiledContract,
		compilation:      compilation,
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

// WithTargetedAssertionMethods filters the assertion test methods to those in the target list.
func (c *Contract) WithTargetedAssertionMethods(target []string) *Contract {
	var candidateMethods []abi.Method
	for _, method := range c.assertionTestMethods {
		canonicalSig := strings.Join([]string{c.name, method.Sig}, ".")
		if containsMethod(target, canonicalSig) {
			candidateMethods = append(candidateMethods, method)
		}
	}
	c.assertionTestMethods = candidateMethods
	return c
}

// WithExcludedAssertionMethods filters the assertion test methods to all methods not in excluded list.
func (c *Contract) WithExcludedAssertionMethods(excludedMethods []string) *Contract {
	var candidateMethods []abi.Method
	for _, method := range c.assertionTestMethods {
		canonicalSig := strings.Join([]string{c.name, method.Sig}, ".")
		if !containsMethod(excludedMethods, canonicalSig) {
			candidateMethods = append(candidateMethods, method)
		}
	}
	c.assertionTestMethods = candidateMethods
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

// PropertyTestMethods returns the methods that are property tests.
func (c *Contract) PropertyTestMethods() []abi.Method {
	return c.propertyTestMethods
}

// OptimizationTestMethods returns the methods that are optimization tests.
func (c *Contract) OptimizationTestMethods() []abi.Method {
	return c.optimizationTestMethods
}

// AssertionTestMethods returns the methods that are assertion tests.
func (c *Contract) AssertionTestMethods() []abi.Method {
	return c.assertionTestMethods
}

func (c *Contract) AddTestMethods(assertion, property, optimization []abi.Method) {
	c.assertionTestMethods = assertion
	c.propertyTestMethods = property
	c.optimizationTestMethods = optimization
}
