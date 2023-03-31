package compilation

import (
	"fmt"
	"github.com/crytic/medusa/compilation/platforms"
)

// defaultPlatformConfigGenerator is a mapping of platform identifier to generator functions which can be used to create
// a default configuration for the given platform. Each platform which provides a generator in this mapping will be
// considered a supported compilation platform for a configs.CompilationConfig. Items are populated in the init method.
var defaultPlatformConfigGenerator map[string]func() platforms.PlatformConfig

// init is called once per inclusion of a package. This method is used on startup to populate
// defaultPlatformConfigGenerator and add supported platforms.
func init() {
	// Define a list of default platform config generators
	generators := []func() platforms.PlatformConfig{
		func() platforms.PlatformConfig { return platforms.NewSolcCompilationConfig("contract.sol") },
		func() platforms.PlatformConfig { return platforms.NewTruffleCompilationConfig(".") },
		func() platforms.PlatformConfig { return platforms.NewCryticCompilationConfig(".") },
	}

	// Initialize our platform config generator.
	defaultPlatformConfigGenerator = make(map[string]func() platforms.PlatformConfig)

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

// GetDefaultPlatformConfig obtains a PlatformConfig from the default generator for the provided platform.
func GetDefaultPlatformConfig(platform string) platforms.PlatformConfig {
	return defaultPlatformConfigGenerator[platform]()
}
