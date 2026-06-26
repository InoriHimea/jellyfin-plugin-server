package storage

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
	"github.com/inorihimea/jellyfin-plugin-server/internal/db"
	"github.com/inorihimea/jellyfin-plugin-server/internal/logger"
)

// CleanResult summarises what a cleanup run did (or would do in dry-run mode).
type CleanResult struct {
	LRURemoved    []string `json:"lru_removed"`
	OrphanRemoved []string `json:"orphan_removed"`
	BytesFreed    int64    `json:"bytes_freed"`
	DryRun        bool     `json:"dry_run"`
}

// RunCleanup performs LRU version eviction and orphan file removal.
// Pass dryRun=true to preview without touching anything.
func RunCleanup(dryRun bool) (CleanResult, error) {
	result := CleanResult{DryRun: dryRun}

	lruFiles, lruBytes, err := lruCleanup(dryRun)
	if err != nil {
		return result, err
	}
	result.LRURemoved = lruFiles
	result.BytesFreed += lruBytes

	orphanFiles, orphanBytes, err := orphanCleanup(dryRun)
	if err != nil {
		return result, err
	}
	result.OrphanRemoved = orphanFiles
	result.BytesFreed += orphanBytes

	if !dryRun && (len(lruFiles)+len(orphanFiles)) > 0 {
		logger.Info("cleanup complete", map[string]any{
			"lru_removed":    len(lruFiles),
			"orphan_removed": len(orphanFiles),
			"bytes_freed":    result.BytesFreed,
		})
		db.WriteLog("INFO", "cleanup complete",
			"lru="+itoa(len(lruFiles))+" orphan="+itoa(len(orphanFiles)))
	}

	return result, nil
}

// lruCleanup keeps the RetainVersions most-recently-downloaded "done" versions
// per plugin and evicts the rest.
func lruCleanup(dryRun bool) (removed []string, bytesFreed int64, err error) {
	retain := config.Get().Storage.KeepVersions
	if retain <= 0 {
		retain = 3
	}

	// Collect all plugins that have more than `retain` done versions.
	rows, err := db.DB.Query(`
		SELECT plugin_id, COUNT(*) as cnt
		FROM plugin_versions
		WHERE download_status = 'done'
		GROUP BY plugin_id
		HAVING cnt > ?`, retain)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var pluginIDs []string
	for rows.Next() {
		var id string
		var cnt int
		if err := rows.Scan(&id, &cnt); err != nil {
			continue
		}
		pluginIDs = append(pluginIDs, id)
	}
	rows.Close()

	for _, pid := range pluginIDs {
		vrows, err := db.DB.Query(`
			SELECT id, COALESCE(local_path, '')
			FROM plugin_versions
			WHERE plugin_id = ? AND download_status = 'done'
			ORDER BY COALESCE(downloaded_at, '') DESC`, pid)
		if err != nil {
			continue
		}

		type vEntry struct{ id, path string }
		var versions []vEntry
		for vrows.Next() {
			var v vEntry
			vrows.Scan(&v.id, &v.path)
			versions = append(versions, v)
		}
		vrows.Close()

		// Skip the first `retain` entries; evict the rest.
		for i := retain; i < len(versions); i++ {
			v := versions[i]
			var size int64
			if v.path != "" {
				if info, err := os.Stat(v.path); err == nil {
					size = info.Size()
				}
				if !dryRun {
					os.Remove(v.path)
				}
				removed = append(removed, v.path)
				bytesFreed += size
			}
			if !dryRun {
				db.DB.Exec(`UPDATE plugin_versions SET download_status='pending', local_path=NULL, downloaded_at=NULL WHERE id=?`, v.id)
			}
		}
	}

	return removed, bytesFreed, nil
}

// orphanCleanup finds files in PackagesDir that are not referenced by any
// plugin_versions.local_path and removes them.
func orphanCleanup(dryRun bool) (removed []string, bytesFreed int64, err error) {
	dir := config.PackagesDir()

	// Build set of known local_path values.
	rows, err := db.DB.Query(`SELECT COALESCE(local_path,'') FROM plugin_versions WHERE local_path IS NOT NULL`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	known := make(map[string]struct{})
	for rows.Next() {
		var p string
		rows.Scan(&p)
		if p != "" {
			known[p] = struct{}{}
		}
	}
	rows.Close()

	err = filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return nil
		}
		// Skip temp files from in-progress downloads.
		if strings.HasSuffix(filepath.Base(path), ".tmp") {
			return nil
		}
		if _, ok := known[path]; !ok {
			bytesFreed += info.Size()
			removed = append(removed, path)
			if !dryRun {
				os.Remove(path)
			}
		}
		return nil
	})

	return removed, bytesFreed, err
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
