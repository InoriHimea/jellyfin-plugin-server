package manifest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/inorihimea/jellyfin-plugin-server/internal/db"
	"github.com/inorihimea/jellyfin-plugin-server/internal/logger"
	proxyClient "github.com/inorihimea/jellyfin-plugin-server/internal/proxy"
	"github.com/google/uuid"
)

// FetchAndStore fetches a manifest from the upstream URL, persists metadata to DB.
// Returns (catalog, changed, error). changed=false means 304 Not Modified.
func FetchAndStore(repoID, repoURL string) (Catalog, bool, error) {
	var etag, lastMod string
	_ = db.DB.QueryRow(
		`SELECT COALESCE(etag,''), COALESCE(last_fetched,'') FROM repos WHERE id=?`, repoID,
	).Scan(&etag, &lastMod)

	resp, err := proxyClient.Get(repoURL, etag, "")
	if err != nil {
		return nil, false, fmt.Errorf("fetch manifest %s: %w", repoURL, err)
	}

	now := db.Now()

	if resp.StatusCode == http.StatusNotModified {
		db.DB.Exec(`UPDATE repos SET last_fetched=? WHERE id=?`, now, repoID)
		logger.Info("manifest not modified", map[string]any{"repo": repoURL})
		return nil, false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("upstream returned %d for %s", resp.StatusCode, repoURL)
	}

	var catalog Catalog
	if err := json.Unmarshal(resp.Body, &catalog); err != nil {
		return nil, false, fmt.Errorf("parse manifest: %w", err)
	}

	if err := persistCatalog(repoID, catalog); err != nil {
		return nil, false, fmt.Errorf("persist manifest: %w", err)
	}

	db.DB.Exec(
		`UPDATE repos SET last_fetched=?, etag=? WHERE id=?`,
		now, resp.ETag, repoID,
	)

	logger.Info("manifest fetched", map[string]any{
		"repo":    repoURL,
		"plugins": len(catalog),
	})
	db.WriteLog("INFO", "manifest fetched", fmt.Sprintf("repo=%s plugins=%d", repoURL, len(catalog)))

	// Trigger background downloads for newly discovered versions.
	go enqueuePending()

	return catalog, true, nil
}

var enqueuePending = func() {}

// SetEnqueueFunc wires the downloader into the fetcher to avoid import cycles.
func SetEnqueueFunc(fn func()) { enqueuePending = fn }

func persistCatalog(repoID string, catalog Catalog) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, p := range catalog {
		pluginID := ""
		_ = tx.QueryRow(`SELECT id FROM plugins WHERE repo_id=? AND guid=?`, repoID, p.GUID).Scan(&pluginID)
		if pluginID == "" {
			pluginID = uuid.NewString()
			_, err = tx.Exec(
				`INSERT INTO plugins (id, repo_id, guid, name, description, overview, owner, category, image_url)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				pluginID, repoID, p.GUID, p.Name, p.Description, p.Overview, p.Owner, p.Category, p.ImageURL,
			)
		} else {
			_, err = tx.Exec(
				`UPDATE plugins SET name=?, description=?, overview=?, owner=?, category=?, image_url=? WHERE id=?`,
				p.Name, p.Description, p.Overview, p.Owner, p.Category, p.ImageURL, pluginID,
			)
		}
		if err != nil {
			return err
		}

		for _, v := range p.Versions {
			versionID := ""
			_ = tx.QueryRow(
				`SELECT id FROM plugin_versions WHERE plugin_id=? AND version=?`, pluginID, v.Version,
			).Scan(&versionID)

			if versionID == "" {
				versionID = uuid.NewString()
				_, err = tx.Exec(
					`INSERT INTO plugin_versions
					 (id, plugin_id, version, changelog, target_abi, source_url, checksum, timestamp, download_status)
					 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
					versionID, pluginID, v.Version, v.ChangeLog, v.TargetABI, v.SourceURL, v.Checksum, v.Timestamp,
				)
			} else {
				// update source URL if it changed
				_, err = tx.Exec(
					`UPDATE plugin_versions SET source_url=?, checksum=?, changelog=? WHERE id=?`,
					v.SourceURL, v.Checksum, v.ChangeLog, versionID,
				)
			}
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// IsTTLExpired returns true when the repo's manifest needs re-fetching.
func IsTTLExpired(lastFetched string, ttlSeconds int) bool {
	if lastFetched == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, lastFetched)
	if err != nil {
		return true
	}
	return time.Since(t) > time.Duration(ttlSeconds)*time.Second
}

// BuildLocalManifest reads the DB and returns a Catalog with sourceUrl replaced by local URLs.
func BuildLocalManifest(repoID, baseURL string) (Catalog, error) {
	rows, err := db.DB.Query(
		`SELECT p.guid, p.name, COALESCE(p.description,''), COALESCE(p.overview,''),
		        COALESCE(p.owner,''), COALESCE(p.category,''), COALESCE(p.image_url,''),
		        v.version, COALESCE(v.changelog,''), COALESCE(v.target_abi,''),
		        v.source_url, v.checksum, COALESCE(v.timestamp,''),
		        COALESCE(v.local_path,''), v.download_status
		 FROM plugins p
		 JOIN plugin_versions v ON v.plugin_id = p.id
		 WHERE p.repo_id = ?
		 ORDER BY p.guid, v.timestamp DESC`, repoID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pluginMap := make(map[string]*Plugin)
	var order []string

	for rows.Next() {
		var (
			guid, name, desc, overview, owner, cat, imageURL string
			ver, changelog, abi, srcURL, checksum, ts        string
			localPath, dlStatus                              string
		)
		if err := rows.Scan(&guid, &name, &desc, &overview, &owner, &cat, &imageURL,
			&ver, &changelog, &abi, &srcURL, &checksum, &ts,
			&localPath, &dlStatus); err != nil {
			return nil, err
		}

		if _, ok := pluginMap[guid]; !ok {
			pluginMap[guid] = &Plugin{
				GUID: guid, Name: name, Description: desc,
				Overview: overview, Owner: owner, Category: cat, ImageURL: imageURL,
			}
			order = append(order, guid)
		}

		resolvedURL := srcURL
		if dlStatus == "done" && localPath != "" {
			resolvedURL = localURL(baseURL, checksum, localPath)
		}

		pluginMap[guid].Versions = append(pluginMap[guid].Versions, Version{
			Version:   ver,
			ChangeLog: changelog,
			TargetABI: abi,
			SourceURL: resolvedURL,
			Checksum:  checksum,
			Timestamp: ts,
		})
	}

	catalog := make(Catalog, 0, len(order))
	for _, g := range order {
		catalog = append(catalog, *pluginMap[g])
	}
	return catalog, nil
}

// BuildUnifiedManifest aggregates all enabled repos into one deduplicated Catalog.
// Per-GUID: metadata from the highest-priority repo; per (GUID, version): highest-priority copy wins.
func BuildUnifiedManifest(baseURL string) (Catalog, error) {
	rows, err := db.DB.Query(
		`SELECT p.guid, p.name, COALESCE(p.description,''), COALESCE(p.overview,''),
		        COALESCE(p.owner,''), COALESCE(p.category,''), COALESCE(p.image_url,''),
		        v.version, COALESCE(v.changelog,''), COALESCE(v.target_abi,''),
		        v.source_url, v.checksum, COALESCE(v.timestamp,''),
		        COALESCE(v.local_path,''), v.download_status
		 FROM plugins p
		 JOIN plugin_versions v ON v.plugin_id = p.id
		 JOIN repos r ON r.id = p.repo_id
		 WHERE r.enabled = 1
		 ORDER BY r.priority DESC, p.guid, v.timestamp DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type pluginEntry struct {
		p            *Plugin
		seenVersions map[string]bool
	}
	seen := make(map[string]*pluginEntry)
	var order []string

	for rows.Next() {
		var (
			guid, name, desc, overview, owner, cat, imageURL string
			ver, changelog, abi, srcURL, checksum, ts        string
			localPath, dlStatus                              string
		)
		if err := rows.Scan(&guid, &name, &desc, &overview, &owner, &cat, &imageURL,
			&ver, &changelog, &abi, &srcURL, &checksum, &ts,
			&localPath, &dlStatus); err != nil {
			return nil, err
		}

		if _, ok := seen[guid]; !ok {
			seen[guid] = &pluginEntry{
				p: &Plugin{
					GUID: guid, Name: name, Description: desc,
					Overview: overview, Owner: owner, Category: cat, ImageURL: imageURL,
				},
				seenVersions: make(map[string]bool),
			}
			order = append(order, guid)
		}

		e := seen[guid]
		if e.seenVersions[ver] {
			continue
		}
		e.seenVersions[ver] = true

		resolvedURL := srcURL
		if dlStatus == "done" && localPath != "" {
			resolvedURL = localURL(baseURL, checksum, localPath)
		}

		e.p.Versions = append(e.p.Versions, Version{
			Version:   ver,
			ChangeLog: changelog,
			TargetABI: abi,
			SourceURL: resolvedURL,
			Checksum:  checksum,
			Timestamp: ts,
		})
	}

	catalog := make(Catalog, 0, len(order))
	for _, g := range order {
		catalog = append(catalog, *seen[g].p)
	}
	return catalog, nil
}

func localURL(base, checksum, localPath string) string {
	// extract filename from localPath
	idx := strings.LastIndex(localPath, "/")
	name := localPath
	if idx >= 0 {
		name = localPath[idx+1:]
	}
	return fmt.Sprintf("%s/plugins/packages/%s/%s", strings.TrimRight(base, "/"), checksum, name)
}
