package pkgcheck

import (
	"strconv"
	"strings"
)

// jellyfinDotnetRuntime maps Jellyfin release series to the major version of
// the .NET runtime that series ships with. A plugin compiled against a .NET
// newer than its host runtime cannot load (assembly refs like
// System.Runtime, Version=10.0 throw NotSupported); older is fine because
// .NET Core 3+ rolls forward minor versions within a major.
//
// Keep this table current when a new Jellyfin series ships.
var jellyfinDotnetRuntime = []struct {
	minor       int // Jellyfin 10.x minor series (e.g. 11 for 10.11)
	dotnetMajor int
}{
	{8, 6},  // 10.8.x → .NET 6
	{9, 8},  // 10.9.x → .NET 8
	{10, 8}, // 10.10.x → .NET 8
	{11, 9}, // 10.11.x → .NET 9
}

// MaxDotnetMajor returns the highest .NET major version a Jellyfin server of
// the given version (e.g. "10.11.5") can load, or 0 when the series is
// unknown — 0 meaning "don't filter", so an outdated mapping never silently
// hides packages for a future Jellyfin we haven't mapped yet.
func MaxDotnetMajor(jellyfinVersion string) int {
	parts := strings.SplitN(strings.TrimPrefix(jellyfinVersion, "v"), ".", 3)
	if len(parts) < 2 {
		return 0
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}

	if major != 10 {
		// Only the 10.x series is mapped. Future majors (11.x) and pre-10
		// releases get 0 = "don't filter", so an outdated mapping never
		// silently hides packages for a runtime we don't know.
		return 0
	}

	best := 0
	for _, e := range jellyfinDotnetRuntime {
		if minor >= e.minor {
			best = e.dotnetMajor
		}
	}
	return best
}
