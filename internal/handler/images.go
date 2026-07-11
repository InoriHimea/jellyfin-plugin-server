package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
	"github.com/inorihimea/jellyfin-plugin-server/internal/db"
	"github.com/inorihimea/jellyfin-plugin-server/internal/logger"
	proxyClient "github.com/inorihimea/jellyfin-plugin-server/internal/proxy"
	"golang.org/x/sync/singleflight"
)

// Bounded concurrency for upstream image fetches. Jellyfin's catalog page
// can request 100-300 images at once; without a cap each one opens its own
// connection to the same host (usually raw.githubusercontent.com), which on
// a slow or throttled network path can stall the whole NAS's connection
// table instead of just the plugin server.
var imgSem = make(chan struct{}, 6)

var imgGroup singleflight.Group

// imgFailCache remembers recently-failed guids so a dead/unreachable image
// URL isn't re-dialed on every single catalog page load — only retried after
// the cooldown expires.
var (
	imgFailMu    sync.Mutex
	imgFailCache = map[string]time.Time{}
)

const imgFailCooldown = 15 * time.Minute

func imgRecentlyFailed(guid string) bool {
	imgFailMu.Lock()
	defer imgFailMu.Unlock()
	t, ok := imgFailCache[guid]
	return ok && time.Since(t) < imgFailCooldown
}

func imgMarkFailed(guid string) {
	imgFailMu.Lock()
	imgFailCache[guid] = time.Now()
	imgFailMu.Unlock()
}

func imgClearFailed(guid string) {
	imgFailMu.Lock()
	delete(imgFailCache, guid)
	imgFailMu.Unlock()
}

// fetchAndCacheImage downloads a plugin image through the proxy-aware
// manifest client, writes it to disk, and returns the bytes. Concurrent
// callers for the same guid share one upstream fetch via singleflight.
func fetchAndCacheImage(guid, upstream string) ([]byte, error) {
	v, err, _ := imgGroup.Do(guid, func() (any, error) {
		cachePath := filepath.Join(config.ImagesDir(), guid)
		if data, err := os.ReadFile(cachePath); err == nil && len(data) > 0 {
			return data, nil
		}

		imgSem <- struct{}{}
		defer func() { <-imgSem }()

		resp, err := proxyClient.GetManifest(upstream, "", "")
		if err != nil || resp.StatusCode != http.StatusOK || len(resp.Body) == 0 {
			imgMarkFailed(guid)
			if err == nil {
				err = fmt.Errorf("upstream returned %d", resp.StatusCode)
			}
			return nil, err
		}

		if mkErr := os.MkdirAll(config.ImagesDir(), 0755); mkErr == nil {
			_ = os.WriteFile(cachePath, resp.Body, 0644)
		}
		imgClearFailed(guid)
		return resp.Body, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]byte), nil
}

// PrewarmImages fetches every plugin image that isn't cached on disk yet,
// bounded by imgSem, so Jellyfin's catalog page never pays the upstream
// fetch cost itself. Safe to call repeatedly (cheap no-op once warm).
func PrewarmImages() {
	rows, err := db.DB.Query(`SELECT DISTINCT guid, image_url FROM plugins WHERE image_url <> ''`)
	if err != nil {
		return
	}
	type job struct{ guid, url string }
	var jobs []job
	for rows.Next() {
		var j job
		if rows.Scan(&j.guid, &j.url) == nil {
			jobs = append(jobs, j)
		}
	}
	rows.Close()

	var wg sync.WaitGroup
	warmed := 0
	var warmedMu sync.Mutex
	for _, j := range jobs {
		cachePath := filepath.Join(config.ImagesDir(), j.guid)
		if data, statErr := os.ReadFile(cachePath); statErr == nil && len(data) > 0 {
			continue
		}
		if imgRecentlyFailed(j.guid) {
			continue
		}
		wg.Add(1)
		go func(guid, url string) {
			defer wg.Done()
			if _, err := fetchAndCacheImage(guid, url); err == nil {
				warmedMu.Lock()
				warmed++
				warmedMu.Unlock()
			}
		}(j.guid, j.url)
	}
	wg.Wait()
	if warmed > 0 {
		logger.Info("image prewarm complete", map[string]any{"warmed": warmed, "total": len(jobs)})
	}
}
