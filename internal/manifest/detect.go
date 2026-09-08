package manifest

import (
	"regexp"
	"sync"

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
	"github.com/inorihimea/jellyfin-plugin-server/internal/db"
	"github.com/inorihimea/jellyfin-plugin-server/internal/logger"
)

// jellyfinUAMatcher extracts the server version from the User-Agent
// Jellyfin sends on every catalog poll, e.g.:
//
//	Jellyfin/10.11.8.0
//	MediaBrowser/10.8.13 (Jellyfin; ...)   (legacy format)
var jellyfinUAMatcher = regexp.MustCompile(`(?:^|[\s(])(?:Jellyfin|MediaBrowser)/(\d+\.\d+\.\d+)`)

// detectMu serialises detect-and-fill so two concurrent catalog polls can't
// both decide the version is unset and race writing the same value.
var detectMu sync.Mutex

// DetectFromUserAgent learns compat.jellyfin_version from a client's
// User-Agent when the admin hasn't configured it. This is what makes the
// runtime-compatibility filtering self-configuring: the server can't see
// which Jellyfin instances poll it any other way, yet the UA of the
// /manifest request declares the exact server version.
//
// An explicit compat.jellyfin_version in config always wins — detection
// never overwrites a manual setting.
func DetectFromUserAgent(ua string) {
	v := extractJellyfinVersion(ua)
	if v == "" {
		return
	}

	detectMu.Lock()
	defer detectMu.Unlock()

	cfg := config.Get()
	if cfg.Compat.JellyfinVersion != "" {
		return // manual config wins
	}

	updated := *cfg
	updated.Compat.JellyfinVersion = v
	if err := config.Update(&updated); err != nil {
		return // persistence failed; keep filtering off rather than half-apply
	}
	invalidateUnifiedCache()
	logOnceDetected(v)
}

func extractJellyfinVersion(ua string) string {
	m := jellyfinUAMatcher.FindStringSubmatch(ua)
	if m == nil {
		return ""
	}
	return m[1]
}

var (
	detectedOnceMu sync.Mutex
	detectedOnce   bool
)

func logOnceDetected(v string) {
	detectedOnceMu.Lock()
	defer detectedOnceMu.Unlock()
	if detectedOnce {
		return
	}
	detectedOnce = true
	logger.Info("jellyfin version auto-detected from User-Agent", map[string]any{"version": v})
	db.WriteLog("INFO", "jellyfin version auto-detected", "version="+v+" (source: User-Agent, config was unset)")
}
