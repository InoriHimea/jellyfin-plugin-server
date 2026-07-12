package downloader

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
	"github.com/inorihimea/jellyfin-plugin-server/internal/db"
	"github.com/inorihimea/jellyfin-plugin-server/internal/logger"
	proxyClient "github.com/inorihimea/jellyfin-plugin-server/internal/proxy"
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

	destPath := filepath.Join(destDir, filename)
	if err := os.Rename(tmpPath, destPath); err != nil {
		markFailed(versionID, fmt.Sprintf("rename: %v", err))
		return err
	}

	db.DB.Exec(
		`UPDATE plugin_versions SET download_status='done', local_path=?, downloaded_at=?, fail_reason='' WHERE id=?`,
		destPath, db.Now(), versionID,
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
	db.DB.Exec(
		`UPDATE plugin_versions SET download_status='failed', fail_reason=? WHERE id=?`, reason, versionID,
	)
	logger.Warn("download failed", map[string]any{"version_id": versionID, "reason": reason})
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
