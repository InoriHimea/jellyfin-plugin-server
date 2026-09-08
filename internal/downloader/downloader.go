package downloader

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
	"github.com/inorihimea/jellyfin-plugin-server/internal/db"
	"github.com/inorihimea/jellyfin-plugin-server/internal/logger"
	proxyClient "github.com/inorihimea/jellyfin-plugin-server/internal/proxy"
	"github.com/inorihimea/jellyfin-plugin-server/internal/pkgcheck"
	"github.com/inorihimea/jellyfin-plugin-server/internal/storage"
	"golang.org/x/sync/singleflight"
)

var (
	sf      singleflight.Group
	sem     chan struct{}
	semOnce sync.Once
	wg      sync.WaitGroup
)

// Wait blocks until all in-flight downloads have completed.
func Wait() { wg.Wait() }

func getSem() chan struct{} {
	semOnce.Do(func() {
		n := config.Get().Cache.MaxConcurrentDL
		if n <= 0 {
			n = 4
		}
		sem = make(chan struct{}, n)
	})
	return sem
}

// Enqueue schedules a background download for a plugin version.
// It returns immediately; the download runs in a goroutine.
func Enqueue(versionID, checksum, sourceURL, filename string) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		sf.Do(checksum, func() (any, error) {
			getSem() <- struct{}{}
			defer func() { <-getSem() }()
			download(versionID, checksum, sourceURL, filename)
			return nil, nil
		})
	}()
}

// EnqueueSync downloads synchronously and returns when complete.
// Used for on-demand fetches triggered by package requests.
func EnqueueSync(versionID, checksum, sourceURL, filename string) error {
	_, err, _ := sf.Do(checksum, func() (any, error) {
		getSem() <- struct{}{}
		defer func() { <-getSem() }()
		return nil, download(versionID, checksum, sourceURL, filename)
	})
	return err
}

func download(versionID, checksum, sourceURL, filename string) error {
	// Enforce disk limit before starting.
	cfg := config.Get()
	if limitMB := cfg.Storage.MaxDiskMB; limitMB > 0 {
		used, err := storage.DiskUsage()
		if err == nil && used > int64(limitMB)*1024*1024 {
			msg := fmt.Sprintf("disk limit %d MB exceeded, skipping %s", limitMB, filename)
			markFailed(versionID, msg)
			logger.Warn("disk limit exceeded", map[string]any{"limit_mb": limitMB, "file": filename})
			db.WriteLog("WARN", "disk limit exceeded", msg)
			return fmt.Errorf("%s", msg)
		}
	}

	db.DB.Exec(
		`UPDATE plugin_versions SET download_status='downloading' WHERE id=?`, versionID,
	)

	destDir := config.PackagesDir()
	if err := os.MkdirAll(destDir, 0755); err != nil {
		markFailed(versionID, fmt.Sprintf("mkdir: %v", err))
		return err
	}

	tmpFile, err := os.CreateTemp(destDir, "dl-*.tmp")
	if err != nil {
		markFailed(versionID, fmt.Sprintf("tmpfile: %v", err))
		return err
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath) // no-op if already renamed
	}()

	// Fetch through the configured proxy (stream client: no body-read timeout).
	// Plain http.Get would bypass the proxy and fail on networks where GitHub
	// is unreachable directly.
	resp, err := fetchWithRetry(sourceURL, 3)
	if err != nil {
		markFailed(versionID, fmt.Sprintf("http get: %v", err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		markFailed(versionID, fmt.Sprintf("upstream %d", resp.StatusCode))
		return fmt.Errorf("upstream returned %d", resp.StatusCode)
	}

	entry := &progressEntry{
		VersionID: versionID,
		Checksum:  checksum,
		Filename:  filename,
		Total:     resp.ContentLength,
		StartedAt: time.Now(),
	}
	progress.Store(versionID, entry)
	defer progress.Delete(versionID)

	h := md5.New()
	w := io.MultiWriter(tmpFile, h, progressWriter{entry})
	if _, err := io.Copy(w, resp.Body); err != nil {
		markFailed(versionID, fmt.Sprintf("copy: %v", err))
		return err
	}
	tmpFile.Close()

	got := fmt.Sprintf("%x", h.Sum(nil))
	want := strings.ToLower(strings.TrimSpace(checksum))
	if want != "" && got != want {
		os.Remove(tmpPath)
		msg := fmt.Sprintf("checksum mismatch: got=%s want=%s", got, want)
		markFailed(versionID, msg)
		logger.Error("checksum mismatch", map[string]any{
			"version_id": versionID, "got": got, "want": want,
		})
		db.WriteLog("ERROR", "checksum mismatch", msg)
		return fmt.Errorf("%s", msg)
	}

	// The checksum only proves the mirror sent what the mirror's manifest
	// declared — mirrors can ship a same-version zip compiled against a
	// newer .NET than the Jellyfin runtime (assembly refs like
	// System.Runtime, Version=10.0 fail NotSupported at load). Scan the
	// package content itself and reject those before they're ever served.
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		markFailed(versionID, fmt.Sprintf("readback: %v", err))
		return err
	}
	dotnetMajor := pkgcheck.ScanZipDotnetMajor(data)
	if allowed := pkgcheck.MaxDotnetMajor(cfg.Compat.JellyfinVersion); allowed > 0 && dotnetMajor > allowed {
		os.Remove(tmpPath)
		msg := fmt.Sprintf("runtime incompatible: package needs .NET %d, Jellyfin %s ships .NET %d",
			dotnetMajor, cfg.Compat.JellyfinVersion, allowed)
		markFailed(versionID, msg)
		logger.Warn("package rejected: runtime incompatible", map[string]any{
			"version_id": versionID, "file": filename,
			"package_dotnet": dotnetMajor, "runtime_dotnet": allowed,
		})
		db.WriteLog("WARN", "package rejected: runtime incompatible", fmt.Sprintf("file=%s %s", filename, msg))
		return fmt.Errorf("%s", msg)
	}

	destPath := filepath.Join(destDir, filename)
	if err := os.Rename(tmpPath, destPath); err != nil {
		markFailed(versionID, fmt.Sprintf("rename: %v", err))
		return err
	}

	db.DB.Exec(
		`UPDATE plugin_versions SET download_status='done', local_path=?, downloaded_at=?, fail_reason='', dotnet_major=? WHERE id=?`,
		destPath, db.Now(), dotnetMajor, versionID,
	)
	logger.Info("package downloaded", map[string]any{"file": filename, "checksum": checksum})
	db.WriteLog("INFO", "package downloaded", fmt.Sprintf("file=%s", filename))
	return nil
}

// fetchWithRetry GETs a URL via the proxy-aware stream client, retrying
// transient failures with a short backoff.
func fetchWithRetry(rawURL string, attempts int) (*http.Response, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		if i > 0 {
			time.Sleep(time.Duration(i) * 2 * time.Second)
		}
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "jellyfin-plugin-server/1.0")
		resp, err := proxyClient.GetStreamClient().Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		// Retry on 5xx / 429; other statuses are returned to the caller.
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = fmt.Errorf("upstream returned %d", resp.StatusCode)
			continue
		}
		return resp, nil
	}
	return nil, lastErr
}

func markFailed(versionID, reason string) {
	if len(reason) > 300 {
		reason = reason[:300]
	}
	status := "failed"
	if isPermanentFailure(reason) {
		// Checksum mismatches and 4xx responses mean the upstream manifest
		// itself is stale (deleted release, renamed asset, wrong declared
		// hash) — retrying downloads the same broken thing again forever.
		// Excluded from EnqueueAllPending/apiRetryFailed by using a distinct
		// status; persistCatalog resets it to 'pending' if the upstream
		// checksum actually changes, so a real fix upstream still recovers.
		status = "failed_permanent"
	}
	db.DB.Exec(
		`UPDATE plugin_versions SET download_status=?, fail_reason=? WHERE id=?`, status, reason, versionID,
	)
	logger.Warn("download failed", map[string]any{"version_id": versionID, "reason": reason, "permanent": status == "failed_permanent"})
}

// isPermanentFailure reports whether a failure reason indicates the
// upstream data itself is wrong (bad checksum, a 4xx meaning the file
// genuinely isn't there, or a package built against an unsupported .NET
// runtime) rather than a transient network/server problem worth
// auto-retrying on the next refresh cycle.
func isPermanentFailure(reason string) bool {
	if strings.HasPrefix(reason, "checksum mismatch") {
		return true
	}
	if strings.HasPrefix(reason, "runtime incompatible") {
		return true
	}
	if rest, ok := strings.CutPrefix(reason, "upstream "); ok {
		if code, err := strconv.Atoi(rest); err == nil {
			return code >= 400 && code < 500
		}
	}
	return false
}

// RecoverStuckDownloads resets any version left in 'downloading' back to
// 'pending'. A process restart kills every in-flight download goroutine, so
// a row still marked 'downloading' at startup is orphaned: EnqueueAllPending
// never re-queues that status, and nothing else will ever touch it again.
// Call this once at boot, before EnqueueAllPending runs.
func RecoverStuckDownloads() {
	res, err := db.DB.Exec(`UPDATE plugin_versions SET download_status='pending' WHERE download_status='downloading'`)
	if err != nil {
		logger.Warn("recover stuck downloads failed", map[string]any{"err": err})
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		logger.Info("recovered stuck downloads", map[string]any{"count": n})
		db.WriteLog("INFO", "recovered stuck downloads", fmt.Sprintf("count=%d", n))
	}
}

// BackfillRuntimeCompat re-scans every already-downloaded package against
// the configured Jellyfin runtime. Versions downloaded before the runtime
// check existed (or before compat.jellyfin_version was set) may be cached
// packages that can't actually load — reclassify those as permanent
// failures and delete the local files so the manifest falls back to a
// compatible source for the same version. A no-op when no Jellyfin version
// is configured.
func BackfillRuntimeCompat() {
	allowed := pkgcheck.MaxDotnetMajor(config.Get().Compat.JellyfinVersion)
	if allowed == 0 {
		return
	}

	rows, err := db.DB.Query(
		`SELECT id, COALESCE(local_path,''), COALESCE(dotnet_major,0) FROM plugin_versions
		 WHERE download_status='done' AND local_path<>''`,
	)
	if err != nil {
		logger.Warn("runtime backfill query failed", map[string]any{"err": err})
		return
	}
	type candidate struct {
		id     string
		path   string
		major  int
	}
	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.path, &c.major); err != nil {
			continue
		}
		candidates = append(candidates, c)
	}
	rows.Close()

	for _, c := range candidates {
		major := c.major
		if major == 0 {
			data, err := os.ReadFile(c.path)
			if err != nil {
				continue // file gone; the fast path in handlePackage already handles that
			}
			major = pkgcheck.ScanZipDotnetMajor(data)
		}
		if major <= allowed {
			continue
		}
		msg := fmt.Sprintf("runtime incompatible: package needs .NET %d, Jellyfin %s ships .NET %d",
			major, config.Get().Compat.JellyfinVersion, allowed)
		db.DB.Exec(
			`UPDATE plugin_versions SET download_status='failed_permanent', fail_reason=?, local_path='', dotnet_major=? WHERE id=?`,
			msg, major, c.id,
		)
		os.Remove(c.path)
		logger.Warn("backfill: cached package is runtime-incompatible", map[string]any{
			"version_id": c.id, "package_dotnet": major, "runtime_dotnet": allowed,
		})
		db.WriteLog("WARN", "backfill: cached package is runtime-incompatible", msg)
	}
}

// EnqueueAllPending enqueues all versions in 'pending' state, plus previously
// 'failed' ones so transient network errors are retried on every refresh cycle.
func EnqueueAllPending() {
	rows, err := db.DB.Query(
		`SELECT id, checksum, source_url FROM plugin_versions
		 WHERE download_status IN ('pending', 'failed')`,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, checksum, srcURL string
		if err := rows.Scan(&id, &checksum, &srcURL); err != nil {
			continue
		}
		filename := filenameFromURL(srcURL, checksum)
		Enqueue(id, checksum, srcURL, filename)
	}
}

func filenameFromURL(rawURL, checksum string) string {
	idx := strings.LastIndex(rawURL, "/")
	if idx >= 0 && idx < len(rawURL)-1 {
		return rawURL[idx+1:]
	}
	return checksum + ".zip"
}
