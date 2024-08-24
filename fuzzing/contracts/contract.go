package contracts

import (
	"strings"

	"golang.org/x/exp/slices"

	"github.com/crytic/medusa/compilation/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// Contracts describes an array of contracts
type Contracts []*Contract

// ContractSetupHook describes a contract setup hook
type ContractSetupHook struct {
	// Method represents the setup function
	Method *abi.Method

	// DeployerAddress represents the fuzzer's deployer address, to be used when calling the setup hook.
	DeployerAddress common.Address
}

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

	// setupHook describes the contract's setup hook, if it exists.
	SetupHook *ContractSetupHook

	// PropertyTestMethods are the methods that are property tests.
	PropertyTestMethods []abi.Method

	// OptimizationTestMethods are the methods that are optimization tests.
	OptimizationTestMethods []abi.Method

	// AssertionTestMethods are ALL other methods that are not property or optimization tests by default.
	// If configured, the methods will be targeted or excluded based on the targetFunctionSignatures
	// and excludedFunctionSignatures, respectively.
	AssertionTestMethods []abi.Method
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

// WithTargetedAssertionMethods filters the assertion test methods to those in the target list.
func (c *Contract) WithTargetedAssertionMethods(target []string) *Contract {
	var candidateMethods []abi.Method
	for _, method := range c.AssertionTestMethods {
		canonicalSig := strings.Join([]string{c.name, method.Sig}, ".")
		if slices.Contains(target, canonicalSig) {
			candidateMethods = append(candidateMethods, method)
		}
	}
	c.AssertionTestMethods = candidateMethods
	return c
}

// WithExcludedAssertionMethods filters the assertion test methods to all methods not in excluded list.
func (c *Contract) WithExcludedAssertionMethods(excludedMethods []string) *Contract {
	var candidateMethods []abi.Method
	for _, method := range c.AssertionTestMethods {
		canonicalSig := strings.Join([]string{c.name, method.Sig}, ".")
		if !slices.Contains(excludedMethods, canonicalSig) {
			candidateMethods = append(candidateMethods, method)
		}
	}
	c.AssertionTestMethods = candidateMethods
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
