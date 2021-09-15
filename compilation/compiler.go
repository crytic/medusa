package compilation

import (
	"encoding/json"
	"fmt"
	"github.com/trailofbits/medusa/compilation/platforms"
	"github.com/trailofbits/medusa/compilation/types"
	"github.com/trailofbits/medusa/configs"
)

var supportedCompilationPlatforms = []string {
	"solc",
	"truffle",
	"hardhat",
	"dapp",
	"brownie",
	"waffle",
}

func GetSupportedCompilationPlatforms() []string {
	return supportedCompilationPlatforms
}

func IsSupportedCompilationPlatform(platform string) bool {
	// Verify the platform is in our supported array
	for _, supportedPlatform := range supportedCompilationPlatforms {
		if platform == supportedPlatform {
			return true
		}
	}
	return false
}

func GetDefaultCompilationConfig(platform string) (*configs.CompilationConfig, error) {
	// Verify the platform is valid
	if !IsSupportedCompilationPlatform(platform) {
		return nil, fmt.Errorf("could not get default compilation configs: platform '%s' is unsupported", platform)
	}

	// Switch on our platform to deserialize our platform compilation configs
	var platformConfig *json.RawMessage
	if platform == "solc" {
		solcConfig := platforms.NewSolcCompilationConfig("contract.sol")
		b, err := json.Marshal(solcConfig)
		if err != nil {
			return nil, err
		}
		platformConfig = (*json.RawMessage)(&b)
	} else if platform == "truffle" {
		truffleConfig := platforms.NewTruffleCompilationConfig(".")
		b, err := json.Marshal(truffleConfig)
		if err != nil {
			return nil, err
		}
		platformConfig = (*json.RawMessage)(&b)
	} else if platform == "hardhat" {
		hardhatConfig := platforms.NewHardhatCompilationConfig(".")
		b, err := json.Marshal(hardhatConfig)
		if err != nil {
			return nil, err
		}
		platformConfig = (*json.RawMessage)(&b)
	} else if platform == "dapp" {
		dappConfig := platforms.NewDappCompilationConfig(".")
		b, err := json.Marshal(dappConfig)
		if err != nil {
			return nil, err
		}
		platformConfig = (*json.RawMessage)(&b)
	} else if platform == "brownie" {
		brownieConfig := platforms.NewBrownieCompilationConfig(".")
		b, err := json.Marshal(brownieConfig)
		if err != nil {
			return nil, err
		}
		platformConfig = (*json.RawMessage)(&b)
	} else if platform == "waffle" {
		waffleConfig := platforms.NewWaffleCompilationConfig(".")
		b, err := json.Marshal(waffleConfig)
		if err != nil {
			return nil, err
		}
		platformConfig = (*json.RawMessage)(&b)
	}

	// Return the compilation configs containing our platform-specific configs
	return &configs.CompilationConfig{Platform: platform, PlatformConfig: platformConfig}, nil
}

func Compile(config configs.CompilationConfig) ([]types.Compilation, string, error) {
	// Verify the platform is valid
	if !IsSupportedCompilationPlatform(config.Platform) {
		return nil, "", fmt.Errorf("could not compile from configs: platform '%s' is unsupported", config.Platform)
	}

	// Switch on our platform to deserialize our platform compilation configs
	if config.Platform == "solc" {
		// Parse a solc config out of the underlying configs
		solcConfig := platforms.SolcCompilationConfig{}
		err := json.Unmarshal(*config.PlatformConfig, &solcConfig)
		if err != nil {
			return nil, "", err
		}

		// Compile using our solc configs
		return solcConfig.Compile()
	} else if config.Platform == "truffle" {
		// Parse a truffle config out of the underlying configs
		truffleConfig := platforms.TruffleCompilationConfig{}
		err := json.Unmarshal(*config.PlatformConfig, &truffleConfig)
		if err != nil {
			return nil, "", err
		}

		// Compile using our solc configs
		return truffleConfig.Compile()
	} else if config.Platform == "hardhat" {
		// Parse a truffle config out of the underlying configs
		hardhatConfig := platforms.HardhatCompilationConfig{}
		err := json.Unmarshal(*config.PlatformConfig, &hardhatConfig)
		if err != nil {
			return nil, "", err
		}

		// Compile using our solc configs
		return hardhatConfig.Compile()
	} else if config.Platform == "dapp" {
		// Parse a truffle config out of the underlying configs
		dappConfig := platforms.DappCompilationConfig{}
		err := json.Unmarshal(*config.PlatformConfig, &dappConfig)
		if err != nil {
			return nil, "", err
		}

		// Compile using our solc configs
		return dappConfig.Compile()
	} else if config.Platform == "brownie" {
		// Parse a truffle config out of the underlying configs
		brownieConfig := platforms.BrownieCompilationConfig{}
		err := json.Unmarshal(*config.PlatformConfig, &brownieConfig)
		if err != nil {
			return nil, "", err
		}

		// Compile using our solc configs
		return brownieConfig.Compile()
	} else if config.Platform == "waffle" {
		// Parse a truffle config out of the underlying configs
		waffleConfig := platforms.WaffleCompilationConfig{}
		err := json.Unmarshal(*config.PlatformConfig, &waffleConfig)
		if err != nil {
			return nil, "", err
		}

		// Compile using our solc configs
		return waffleConfig.Compile()
	}

	// Panic if we didn't handle some other case. This should not be hit unless developer error occurs.
	panic(fmt.Sprintf("platform '%s' is marked supported but is not implemented", config.Platform))
}
