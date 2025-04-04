package config

import (
	"github.com/crytic/medusa-geth/common"
	"github.com/crytic/medusa-geth/core/vm"
)

// TestChainConfig represents the chain configuration.
type TestChainConfig struct {
	// CodeSizeCheckDisabled indicates whether code size checks should be disabled in the EVM. This allows for code
	// size to be disabled without disabling the entire EIP it was introduced.
	CodeSizeCheckDisabled bool `json:"codeSizeCheckDisabled"`

	// CheatCodeConfig indicates the configuration for EVM cheat codes to use.
	CheatCodeConfig CheatCodeConfig `json:"cheatCodes"`

	// SkipAccountChecks skips account pre-checks like nonce validation and disallowing non-EOA tx senders (this is done in eth_call, for instance).
	SkipAccountChecks bool `json:"skipAccountChecks"`

	// ContractAddressOverrides describes contracts that are going to be deployed at deterministic addresses
	ContractAddressOverrides map[common.Hash]common.Address `json:"contractAddressOverrides,omitempty"`

	// ForkConfig indicates the RPC configuration if fuzzing using a network fork.
	ForkConfig ForkConfig `json:"forkConfig,omitempty"`
}

// ForkConfig describes configuration for fuzzing using a network fork
type ForkConfig struct {
	ForkModeEnabled bool   `json:"forkModeEnabled"`
	RpcUrl          string `json:"rpcUrl"`
	RpcBlock        uint64 `json:"rpcBlock"`
	PoolSize        uint   `json:"poolSize"`
}

// CheatCodeConfig describes any configuration options related to the use of vm extensions (a.k.a. cheat codes)
type CheatCodeConfig struct {
	// CheatCodesEnabled indicates whether cheat code pre-compiles should be enabled in the chain.
	CheatCodesEnabled bool `json:"cheatCodesEnabled"`

	// EnableFFI describes whether the FFI cheat code should be enabled. Enablement allows for arbitrary code execution
	// on the tester's machine
	EnableFFI bool `json:"enableFFI"`
}

// GetVMConfigExtensions derives a vm.ConfigExtensions from the provided TestChainConfig.
func (t *TestChainConfig) GetVMConfigExtensions() *vm.ConfigExtensions {
	// Create a copy of the contract address overrides that can be ephemerally updated by medusa-geth
	contractAddressOverrides := make(map[common.Hash]common.Address)
	for hash, addr := range t.ContractAddressOverrides {
		contractAddressOverrides[hash] = addr
	}

	// Obtain our vm config extensions data structure
	return &vm.ConfigExtensions{
		OverrideCodeSizeCheck:    t.CodeSizeCheckDisabled,
		AdditionalPrecompiles:    make(map[common.Address]vm.PrecompiledContract),
		ContractAddressOverrides: contractAddressOverrides,
	}
}
