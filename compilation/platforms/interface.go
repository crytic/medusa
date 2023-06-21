package platforms

import "github.com/crytic/medusa/compilation/types"

// PlatformConfig describes the interface all compilation platform configs must implement.
type PlatformConfig interface {
	Compile() ([]types.Compilation, string, error)
	Platform() string
	GetTarget() string
	SetTarget(string)
}
