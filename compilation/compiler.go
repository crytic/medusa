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
	}

	// Return the compilation configs containing our platform-specific configs
	return &configs.CompilationConfig{Platform: platform, PlatformConfig: platformConfig}, nil
}

func Compile(config configs.CompilationConfig) ([]types.Compilation, error) {
	// Verify the platform is valid
	if !IsSupportedCompilationPlatform(config.Platform) {
		return nil, fmt.Errorf("could not compile from configs: platform '%s' is unsupported", config.Platform)
	}

	// Switch on our platform to deserialize our platform compilation configs
	if config.Platform == "solc" {
		// Parse a solc config out of the underlying configs
		solcConfig := platforms.SolcCompilationConfig{}
		err := json.Unmarshal(*config.PlatformConfig, &solcConfig)
		if err != nil {
			return nil, err
		}

		// Compile using our solc configs
		return solcConfig.Compile()
	} else if config.Platform == "truffle" {
		// Parse a truffle config out of the underlying configs
		truffleConfig := platforms.TruffleCompilationConfig{}
		err := json.Unmarshal(*config.PlatformConfig, &truffleConfig)
		if err != nil {
			return nil, err
		}

		// Compile using our solc configs
		return truffleConfig.Compile()
	}

	// Panic if we didn't handle some other case. This should not be hit unless developer error occurs.
	panic(fmt.Sprintf("platform '%s' is marked supported but is not implemented", config.Platform))
}
