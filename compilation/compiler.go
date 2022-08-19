package compilation

import (
	"encoding/json"
	"fmt"
	"github.com/trailofbits/medusa/compilation/platforms"
	"github.com/trailofbits/medusa/compilation/types"
	"github.com/trailofbits/medusa/configs"
)

// defaultPlatformConfigGenerator is a mapping of platform identifier to generator functions which can be used to create
// a default configuration for the given platform. Each platform which provides a generator in this mapping will be
// considered a supported compilation platform for a configs.CompilationConfig. Items are populated in the init method.
var defaultPlatformConfigGenerator map[string]func() platforms.PlatformConfigInterface

// init is called once per inclusion of a package. This method is used on startup to populate
// defaultPlatformConfigGenerator and add supported platforms.
func init() {
	// Define a list of default platform config generators
	generators := []func() platforms.PlatformConfigInterface{
		func() platforms.PlatformConfigInterface { return platforms.NewSolcCompilationConfig("contract.sol") },
		func() platforms.PlatformConfigInterface { return platforms.NewTruffleCompilationConfig(".") },
	}

	// Initialize our platform config generator.
	defaultPlatformConfigGenerator = make(map[string]func() platforms.PlatformConfigInterface)

	// Generate each type of interface to create a mapping for their platform identifiers.
	for _, generator := range generators {
		// Generate a default config and obtain the platform id for it.
		platformConfig := generator()
		platformId := platformConfig.Platform()

		// If this platform already exists in our mapping, panic. Each platform should have a unique identifier.
		if _, platformIdExists := defaultPlatformConfigGenerator[platformId]; platformIdExists {
			panic(fmt.Errorf("the compilation platform '%s' is registered with more than one provider", platformId))
		}

		// Add this entry to our mapping
		defaultPlatformConfigGenerator[platformId] = generator
	}
}

// GetSupportedCompilationPlatforms obtains a list of strings which represent platform identifiers supported by methods
// in this package.
func GetSupportedCompilationPlatforms() []string {
	// Loop through all the platform keys for our generator and add them to a list
	platformIds := make([]string, len(defaultPlatformConfigGenerator))
	i := 0
	for k := range defaultPlatformConfigGenerator {
		platformIds[i] = k
		i++
	}
	return platformIds
}

// IsSupportedCompilationPlatform returns a boolean status indicating if a platform identifier is supported within this
// package.
func IsSupportedCompilationPlatform(platform string) bool {
	// Verify the platform is in our supported map
	_, ok := defaultPlatformConfigGenerator[platform]
	return ok
}

// GetCompilationConfigFromPlatformConfig takes a platforms.PlatformConfigInterface and wraps it in a generic
// configs.CompilationConfig. This allows many platform config types to be serialized/deserialized to their appropriate
// types and supported generally.
func GetCompilationConfigFromPlatformConfig(platformConfig platforms.PlatformConfigInterface) (*configs.CompilationConfig, error) {
	// Marshal our config to a raw message
	b, err := json.Marshal(platformConfig)
	if err != nil {
		return nil, err
	}
	platformConfigMsg := (*json.RawMessage)(&b)

	// Return the compilation configs containing our platform-specific configs
	return &configs.CompilationConfig{Platform: platformConfig.Platform(), PlatformConfig: platformConfigMsg}, nil
}

// GetDefaultCompilationConfig returns a configs.CompilationConfig with default values for a given platform identifier.
// If an error occurs, it is returned instead.
func GetDefaultCompilationConfig(platform string) (*configs.CompilationConfig, error) {
	// Verify the platform is valid
	if !IsSupportedCompilationPlatform(platform) {
		return nil, fmt.Errorf("could not get default compilation configs: platform '%s' is unsupported", platform)
	}

	// Switch on our platform to deserialize our platform compilation configs
	platformConfig := defaultPlatformConfigGenerator[platform]()
	return GetCompilationConfigFromPlatformConfig(platformConfig)
}

// Compile takes a generic configs.CompilationConfig and deserializes the inner platforms.PlatformConfigInterface, which
// is then used to compile the underlying targets. Returns a list of compilations returned by the platform provider or
// an error. Command-line input may also be returned in either case.,
func Compile(config configs.CompilationConfig) ([]types.Compilation, string, error) {
	// Verify the platform is valid
	if !IsSupportedCompilationPlatform(config.Platform) {
		return nil, "", fmt.Errorf("could not compile from configs: platform '%s' is unsupported", config.Platform)
	}

	// Allocate a platform config given our platform string in our compilation config
	// It is necessary to do so as json.Unmarshal needs a concrete structure to populate
	platformConfig := defaultPlatformConfigGenerator[config.Platform]()
	err := json.Unmarshal(*config.PlatformConfig, &platformConfig)
	if err != nil {
		return nil, "", err
	}

	// Compile using our solc configs
	return platformConfig.(platforms.PlatformConfigInterface).Compile()
}
