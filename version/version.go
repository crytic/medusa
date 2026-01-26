// Package version provides build and version information for medusa.
// It uses Go's runtime/debug.ReadBuildInfo() to extract VCS metadata
// embedded at build time (Go 1.18+).
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

// These variables can be set via ldflags at build time for explicit versioning.
// If not set, they will be populated from runtime/debug.ReadBuildInfo().
var (
	// Version is the semantic version of the build.
	Version = "1.4.1"
	// GitCommit is the git commit hash.
	GitCommit = ""
	// GitCommitTime is the timestamp of the git commit.
	GitCommitTime = ""
	// GitTreeDirty indicates if the git tree was dirty at build time.
	GitTreeDirty = ""
)

// Info contains the full version information for the build.
type Info struct {
	Version       string
	GitCommit     string
	GitCommitTime string
	GitTreeDirty  bool
	GoVersion     string
}

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	// Extract VCS info from build settings
	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs.revision":
			if GitCommit == "" {
				GitCommit = kv.Value
			}
		case "vcs.time":
			if GitCommitTime == "" {
				GitCommitTime = kv.Value
			}
		case "vcs.modified":
			if GitTreeDirty == "" {
				GitTreeDirty = kv.Value
			}
		}
	}
}

// GetInfo returns the complete version information.
func GetInfo() Info {
	return Info{
		Version:       Version,
		GitCommit:     GitCommit,
		GitCommitTime: GitCommitTime,
		GitTreeDirty:  GitTreeDirty == "true",
		GoVersion:     runtime.Version(),
	}
}

// ShortCommit returns the first 7 characters of the git commit hash.
func (i Info) ShortCommit() string {
	if len(i.GitCommit) >= 7 {
		return i.GitCommit[:7]
	}
	return i.GitCommit
}

// FormattedTime returns the commit time in a human-readable format.
func (i Info) FormattedTime() string {
	if i.GitCommitTime == "" {
		return "unknown"
	}
	t, err := time.Parse(time.RFC3339, i.GitCommitTime)
	if err != nil {
		return i.GitCommitTime
	}
	return t.Format("2006-01-02 15:04:05 MST")
}

// String returns a formatted multi-line version string.
func (i Info) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("medusa version %s\n", i.Version))

	if i.GitCommit != "" {
		commit := i.ShortCommit()
		if i.GitTreeDirty {
			commit += "-dirty"
		}
		sb.WriteString(fmt.Sprintf("  Commit:     %s\n", commit))
	}

	if i.GitCommitTime != "" {
		sb.WriteString(fmt.Sprintf("  Built:      %s\n", i.FormattedTime()))
	}

	sb.WriteString(fmt.Sprintf("  Go version: %s\n", i.GoVersion))

	return sb.String()
}

// Short returns a single-line version string suitable for --version output.
func (i Info) Short() string {
	v := i.Version
	if i.GitCommit != "" {
		v += "+" + i.ShortCommit()
		if i.GitTreeDirty {
			v += "-dirty"
		}
	}
	return v
}
