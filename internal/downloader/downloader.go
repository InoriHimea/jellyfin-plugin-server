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

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
	"github.com/inorihimea/jellyfin-plugin-server/internal/db"
	"github.com/inorihimea/jellyfin-plugin-server/internal/logger"
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

	resp, err := http.Get(sourceURL)
	if err != nil {
		markFailed(versionID, fmt.Sprintf("http get: %v", err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		markFailed(versionID, fmt.Sprintf("upstream %d", resp.StatusCode))
		return fmt.Errorf("upstream returned %d", resp.StatusCode)
	}

	h := md5.New()
	w := io.MultiWriter(tmpFile, h)
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
		`UPDATE plugin_versions SET download_status='done', local_path=?, downloaded_at=? WHERE id=?`,
		destPath, db.Now(), versionID,
	)
	logger.Info("package downloaded", map[string]any{"file": filename, "checksum": checksum})
	db.WriteLog("INFO", "package downloaded", fmt.Sprintf("file=%s", filename))
	return nil
}

func markFailed(versionID, reason string) {
	db.DB.Exec(
		`UPDATE plugin_versions SET download_status='failed' WHERE id=?`, versionID,
	)
	logger.Warn("download failed", map[string]any{"version_id": versionID, "reason": reason})
}

// EnqueueAllPending enqueues all versions currently in 'pending' state.
func EnqueueAllPending() {
	rows, err := db.DB.Query(
		`SELECT id, checksum, source_url FROM plugin_versions WHERE download_status='pending'`,
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
