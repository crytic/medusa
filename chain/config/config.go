package config

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

// TestChainConfig represents the chain configuration.
type TestChainConfig struct {
	// CodeSizeCheckDisabled indicates whether code size checks should be disabled in the EVM. This allows for code
	// size to be disabled without disabling the entire EIP it was introduced.
	CodeSizeCheckDisabled bool `json:"codeSizeCheckDisabled"`

	// CheatCodeConfig indicates the configuration for EVM cheat codes to use.
	CheatCodeConfig CheatCodeConfig `json:"cheatCodes"`
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
	// Obtain our cheat code precompiled contracts.
	return &vm.ConfigExtensions{
		OverrideCodeSizeCheck: t.CodeSizeCheckDisabled,
		AdditionalPrecompiles: make(map[common.Address]vm.PrecompiledContract),
	}
}
