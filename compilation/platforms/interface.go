package platforms

import "github.com/trailofbits/medusa/compilation/types"

// PlatformConfigInterface describes the interface all compilation platform configs must implement.
type PlatformConfigInterface interface {
	Compile() ([]types.Compilation, string, error)
	Platform() string
}
