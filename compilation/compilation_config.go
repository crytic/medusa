package compilation

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/crytic/medusa/compilation/platforms"
	"github.com/crytic/medusa/compilation/types"
)

// CompilationConfig describes the configuration options used to compile a smart contract
// target.
type CompilationConfig struct {
	// Platform references an identifier indicating which compilation platform to use.
	// PlatformConfig is a structure dependent on the defined Platform.
	Platform string `json:"platform"`

	// PlatformConfig describes the Platform-specific configuration needed to compile.
	PlatformConfig *json.RawMessage `json:"platformConfig"`
}

// NewCompilationConfig returns a CompilationConfig with default values for a given platform identifier.
// If an error occurs, it is returned instead.
func NewCompilationConfig(platform string) (*CompilationConfig, error) {
	// Verify the platform is valid
	if !IsSupportedCompilationPlatform(platform) {
		return nil, fmt.Errorf("could not get default compilation configs: platform '%s' is unsupported", platform)
	}

	// Switch on our platform to deserialize our platform compilation configs
	platformConfig := GetDefaultPlatformConfig(platform)
	return NewCompilationConfigFromPlatformConfig(platformConfig)
}

// NewCompilationConfigFromPlatformConfig takes a platforms.PlatformConfig and wraps it in a generic
// CompilationConfig. This allows many platform config types to be serialized/deserialized to their appropriate
// types and supported generally.
func NewCompilationConfigFromPlatformConfig(platformConfig platforms.PlatformConfig) (*CompilationConfig, error) {
	// Marshal our config to a raw message
	b, err := json.Marshal(platformConfig)
	if err != nil {
		return nil, err
	}
	platformConfigMsg := (*json.RawMessage)(&b)

	// Return the compilation configs containing our platform-specific configs
	return &CompilationConfig{Platform: platformConfig.Platform(), PlatformConfig: platformConfigMsg}, nil
}

// Compile takes a generic CompilationConfig and deserializes the inner platforms.PlatformConfig, which
// is then used to compile the underlying targets. Returns a list of compilations returned by the platform provider or
// an error. Command-line input may also be returned in either case.,
func (c *CompilationConfig) Compile() ([]types.Compilation, string, error) {
	// Get the platform config
	platformConfig, err := c.GetPlatformConfig()
	if err != nil {
		return nil, "", err
	}

	// Compile using our solc configs
	return platformConfig.Compile()
}

// GetPlatformConfig will return the de-serialized version of platforms.PlatformConfig for a given CompilationConfig
func (c *CompilationConfig) GetPlatformConfig() (platforms.PlatformConfig, error) {
	// Ensure that the platform is non-empty
	if c.Platform == "" {
		return nil, fmt.Errorf("must specify a platform for compilation")
	}

	// Ensure that the platform is supported
	if !IsSupportedCompilationPlatform(c.Platform) {
		return nil, fmt.Errorf("compilation platform '%v' is unsupported", c.Platform)
	}

	// Allocate a platform config given our platform string in our compilation config
	// It is necessary to do so as json.Unmarshal needs a concrete structure to populate
	platformConfig := GetDefaultPlatformConfig(c.Platform)
	err := json.Unmarshal(*c.PlatformConfig, &platformConfig)
	if err != nil {
		return nil, err
	}
	return platformConfig, nil
}

// SetPlatformConfig replaces the current platform config with the one provided as an argument
// Note that non-nil platform configs will not be accepted
func (c *CompilationConfig) SetPlatformConfig(platformConfig platforms.PlatformConfig) error {
	// No nil values allowed
	if platformConfig == nil {
		return errors.New("platformConfig must be non-nil")
	}

	// Update platform, assuming the platform is supported
	if !IsSupportedCompilationPlatform(platformConfig.Platform()) {
		return fmt.Errorf("compilation platform '%v' is unsupported", platformConfig.Platform())
	}
	c.Platform = platformConfig.Platform()

	// Serialize
	b, err := json.Marshal(platformConfig)
	if err != nil {
		return err
	}

	// Update the compilation config
	platformConfigMsg := (*json.RawMessage)(&b)
	c.PlatformConfig = platformConfigMsg

	return nil
}

// SetTarget will update the compilation target in the underlying PlatformConfig
func (c *CompilationConfig) SetTarget(newTarget string) error {
	// De-serialize platform config
	platformConfig, err := c.GetPlatformConfig()
	if err != nil {
		return err
	}

	// Update target
	platformConfig.SetTarget(newTarget)

	// Update this config's platformConfig
	err = c.SetPlatformConfig(platformConfig)
	if err != nil {
		return err
	}

	return nil
}
