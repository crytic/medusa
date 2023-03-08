package compilation

import (
	"encoding/json"
	"github.com/pkg/errors"

	"github.com/trailofbits/medusa/compilation/platforms"
	"github.com/trailofbits/medusa/compilation/types"
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
		err := errors.Errorf("could not get default compilation config: platform '%s' is unsupported", platform)
		return nil, errors.WithStack(err)
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
		return nil, errors.WithStack(err)
	}
	platformConfigMsg := (*json.RawMessage)(&b)

	// Return the compilation configs containing our platform-specific configs
	return &CompilationConfig{Platform: platformConfig.Platform(), PlatformConfig: platformConfigMsg}, nil
}

// Compile takes a generic CompilationConfig and deserializes the inner platforms.PlatformConfig, which
// is then used to compile the underlying targets. Returns a list of compilations returned by the platform provider or
// an error. Command-line input may also be returned in either case.,
func (c *CompilationConfig) Compile() ([]types.Compilation, string, error) {
	// Verify the platform is valid
	if !IsSupportedCompilationPlatform(c.Platform) {
		err := errors.Errorf("could not get default compilation config: platform '%s' is unsupported", c.Platform)
		return nil, "", errors.WithStack(err)
	}

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
	// Allocate a platform config given our platform string in our compilation config
	// It is necessary to do so as json.Unmarshal needs a concrete structure to populate
	platformConfig := GetDefaultPlatformConfig(c.Platform)
	err := json.Unmarshal(*c.PlatformConfig, &platformConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return platformConfig, nil
}

// SetPlatformConfig replaces the current platform config with the one provided as an argument
// Note that non-nil platform configs will not be accepted
func (c *CompilationConfig) SetPlatformConfig(platformConfig platforms.PlatformConfig) error {
	// No nil values allowed
	if platformConfig == nil {
		err := errors.Errorf("platformConfig must be non-nil")
		return errors.WithStack(err)
	}

	// Update platform
	c.Platform = platformConfig.Platform()

	// Serialize
	b, err := json.Marshal(platformConfig)
	if err != nil {
		return errors.WithStack(err)
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
