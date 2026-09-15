package manifest

import "regexp"

// jellyfinUAMatcher extracts the server version from the User-Agent
// Jellyfin sends on every catalog poll, e.g.:
//
//	Jellyfin/12.0.0.0
//	MediaBrowser/10.8.13 (Jellyfin; ...)   (legacy format)
var jellyfinUAMatcher = regexp.MustCompile(`(?:^|[\s(])(?:Jellyfin|MediaBrowser)/(\d+\.\d+\.\d+)`)

// JellyfinVersionFromUserAgent returns the Jellyfin version declared by a
// manifest request's User-Agent, or "" when the client is not recognizable.
// It is deliberately request-scoped: a caller must never persist this value
// as the global compatibility default because multiple Jellyfin versions can
// share the same plugin server.
func JellyfinVersionFromUserAgent(ua string) string {
	m := jellyfinUAMatcher.FindStringSubmatch(ua)
	if m == nil {
		return ""
	}
	return m[1]
}
