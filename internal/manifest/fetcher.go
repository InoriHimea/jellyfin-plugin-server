package manifest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/inorihimea/jellyfin-plugin-server/internal/db"
	"github.com/inorihimea/jellyfin-plugin-server/internal/logger"
	proxyClient "github.com/inorihimea/jellyfin-plugin-server/internal/proxy"
	"github.com/google/uuid"
)

// unifiedCache holds the last BuildUnifiedManifest result per baseURL.
// Invalidated whenever FetchAndStore writes new data.
var (
	unifiedMu      sync.RWMutex
	unifiedEntries = map[string]unifiedEntry{}
	unifiedTTL     = 60 * time.Second
)

type unifiedEntry struct {
	catalog   Catalog
	expiresAt time.Time
}

func invalidateUnifiedCache() {
	unifiedMu.Lock()
	unifiedEntries = map[string]unifiedEntry{}
	unifiedMu.Unlock()
}

// FetchAndStore fetches a manifest from the upstream URL, persists metadata to DB.
// Returns (catalog, changed, error). changed=false means 304 Not Modified.
func FetchAndStore(repoID, repoURL string) (Catalog, bool, error) {
	var etag, lastMod string
	_ = db.DB.QueryRow(
		`SELECT COALESCE(etag,''), COALESCE(last_fetched,'') FROM repos WHERE id=?`, repoID,
	).Scan(&etag, &lastMod)

	resp, err := proxyClient.GetManifest(repoURL, etag, "")
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

	// Trigger background image prewarming so Jellyfin's catalog page never
	// pays the upstream image-fetch cost itself.
	go prewarmImages()

	// Invalidate the in-memory unified manifest cache so the next request rebuilds it.
	invalidateUnifiedCache()

	return catalog, true, nil
}

var enqueuePending = func() {}
var prewarmImages = func() {}

// SetEnqueueFunc wires the downloader into the fetcher to avoid import cycles.
func SetEnqueueFunc(fn func()) { enqueuePending = fn }

// SetImagePrewarmFunc wires the image cache into the fetcher to avoid import cycles.
func SetImagePrewarmFunc(fn func()) { prewarmImages = fn }

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
			// Keyed on (version, target_abi): some repos publish multiple ABI-targeted
			// builds under the same version number (e.g. 10.10/10.11 compat builds).
			// Matching on version alone collapses those into one row and silently
			// discards the rest.
			versionID := ""
			var existingChecksum, existingStatus string
			_ = tx.QueryRow(
				`SELECT id, checksum, download_status FROM plugin_versions WHERE plugin_id=? AND version=? AND target_abi=?`,
				pluginID, v.Version, v.TargetABI,
			).Scan(&versionID, &existingChecksum, &existingStatus)

			if versionID == "" {
				versionID = uuid.NewString()
				_, err = tx.Exec(
					`INSERT INTO plugin_versions
					 (id, plugin_id, version, changelog, target_abi, source_url, checksum, timestamp, download_status)
					 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
					versionID, pluginID, v.Version, v.ChangeLog, v.TargetABI, v.SourceURL, v.Checksum, v.Timestamp,
				)
			} else if existingStatus == "failed_permanent" && existingChecksum != v.Checksum {
				// Upstream published a different file under the same
				// version/ABI (fixed a bad checksum, restored a deleted
				// release) — give it a fresh attempt instead of leaving it
				// stuck on the old permanent-failure verdict forever.
				_, err = tx.Exec(
					`UPDATE plugin_versions SET source_url=?, checksum=?, changelog=?, download_status='pending', fail_reason='' WHERE id=?`,
					v.SourceURL, v.Checksum, v.ChangeLog, versionID,
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
				Overview: overview, Owner: owner,
				Category: normalizeCategory(cat), ImageURL: imageProxyURL(baseURL, guid, imageURL),
			}
			order = append(order, guid)
		}

		// Always use our server URL — handlePackage streams from upstream if not cached.
		resolvedURL := localURL(baseURL, checksum, localPath, srcURL)

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
		sortVersionsDesc(pluginMap[g].Versions)
		tagAmbiguousChangelogs(pluginMap[g].Versions)
		catalog = append(catalog, *pluginMap[g])
	}
	return catalog, nil
}

// BuildUnifiedManifest aggregates all enabled repos into one deduplicated Catalog.
// Per-GUID: metadata from the highest-priority repo; per (GUID, version): highest-priority copy wins.
// Results are cached in memory for unifiedTTL (60s) to avoid repeated DB scans.
func BuildUnifiedManifest(baseURL string) (Catalog, error) {
	unifiedMu.RLock()
	entry, ok := unifiedEntries[baseURL]
	unifiedMu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		return entry.catalog, nil
	}

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
					Overview: overview, Owner: owner,
					Category: normalizeCategory(cat), ImageURL: imageProxyURL(baseURL, guid, imageURL),
				},
				seenVersions: make(map[string]bool),
			}
			order = append(order, guid)
		} else if imageURL != "" && seen[guid].p.ImageURL == "" {
			// Higher-priority repo had no image; use the first non-empty one we find.
			seen[guid].p.ImageURL = imageProxyURL(baseURL, guid, imageURL)
		}

		e := seen[guid]
		// Key on (version, targetAbi): different repos may legitimately publish
		// different builds under the same version number for different ABI
		// targets (e.g. a 10.10 compat build vs a 10.11 build). Deduping on
		// version alone would let a higher-priority repo's same-numbered but
		// different release silently shadow a lower-priority repo's build.
		versionKey := ver + "|" + abi
		if e.seenVersions[versionKey] {
			continue
		}
		e.seenVersions[versionKey] = true

		// Always use our server URL — handlePackage streams from upstream if not cached.
		resolvedURL := localURL(baseURL, checksum, localPath, srcURL)

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
		sortVersionsDesc(seen[g].p.Versions)
		tagAmbiguousChangelogs(seen[g].p.Versions)
		catalog = append(catalog, *seen[g].p)
	}

	unifiedMu.Lock()
	unifiedEntries[baseURL] = unifiedEntry{catalog: catalog, expiresAt: time.Now().Add(unifiedTTL)}
	unifiedMu.Unlock()

	return catalog, nil
}

// sortVersionsDesc orders a plugin's versions newest-first by numeric
// comparison of the version string, not by the order rows were collected
// in. BuildUnifiedManifest gathers a plugin's versions repo-by-repo
// (highest priority first), so without this every version from a
// lower-priority repo — however new — would sort after all of a
// higher-priority repo's versions instead of interleaving by recency.
// Ties (same version, different targetAbi — e.g. separate 10.10/10.11
// builds) break by timestamp, newest first.
func sortVersionsDesc(versions []Version) {
	sort.SliceStable(versions, func(i, j int) bool {
		if c := CompareVersionStrings(versions[i].Version, versions[j].Version); c != 0 {
			return c > 0
		}
		return versions[i].Timestamp > versions[j].Timestamp
	})
}

// CompareVersionStrings compares two dot-separated numeric version strings
// component-by-component, up to 4 parts — matching how .NET's
// System.Version (what Jellyfin parses "version" as) compares versions.
// Missing or non-numeric components count as 0. Returns >0 if a>b, <0 if
// a<b, 0 if equal. Exported so other packages (e.g. the catalog API,
// which needs the same "which version is actually newest" logic across
// repos) don't reimplement it.
func CompareVersionStrings(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < 4; i++ {
		var na, nb int
		if i < len(pa) {
			na, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			nb, _ = strconv.Atoi(pb[i])
		}
		if na != nb {
			return na - nb
		}
	}
	return 0
}

// tagAmbiguousChangelogs prefixes the changelog with "[ABI x.y.z.w] " for
// any version number that appears more than once in the list (i.e. the same
// version published as separate builds for different Jellyfin ABI targets,
// like trakt-ex's 10.10/10.11 compat pairs). Jellyfin's own plugin-history
// UI shows only the version number and changelog text — not targetAbi — so
// two entries with an identical version and changelog are otherwise
// indistinguishable to whoever is deciding which one to install. Versions
// that are the only entry for their version number are left untouched.
func tagAmbiguousChangelogs(versions []Version) {
	counts := make(map[string]int, len(versions))
	for _, v := range versions {
		counts[v.Version]++
	}
	for i, v := range versions {
		if counts[v.Version] > 1 && v.TargetABI != "" {
			versions[i].ChangeLog = fmt.Sprintf("[ABI %s] %s", v.TargetABI, v.ChangeLog)
		}
	}
}

// localURL builds the URL our server uses to serve (or proxy) a plugin file.
// It always points to /plugins/packages/{checksum}/{name} so that all downloads
// go through our server regardless of whether the file has been cached locally.
// normalizeCategory maps community-manifest category strings to the official
// Jellyfin category values used in repo.jellyfin.org/master/plugin/manifest.json.
// Jellyfin 10.9+ recognises: General, MoviesAndShows, Administration, LiveTV,
// Subtitles, Music, Books, Anime. Unknown values are mapped to General.
func normalizeCategory(cat string) string {
	switch strings.ToLower(strings.TrimSpace(cat)) {
	case "general", "":
		return "General"
	case "moviesandshows", "movies", "shows", "metadata":
		return "MoviesAndShows"
	case "administration", "admin", "notifications", "notification",
		"authentication", "auth":
		return "Administration"
	case "livetv", "live tv", "live-tv":
		return "LiveTV"
	case "subtitles", "subtitle":
		return "Subtitles"
	case "music":
		return "Music"
	case "books", "book":
		return "Books"
	case "anime", "animation":
		return "Anime"
	default:
		return "General"
	}
}

// imageProxyURL rewrites a plugin image to our /plugins/images/{guid} endpoint
// so clients fetch it from us (proxied + disk-cached) instead of upstream hosts.
func imageProxyURL(base, guid, upstream string) string {
	if upstream == "" {
		return ""
	}
	return fmt.Sprintf("%s/plugins/images/%s", strings.TrimRight(base, "/"), guid)
}

func localURL(base, checksum, localPath, srcURL string) string {
	name := checksum + ".zip"
	if localPath != "" {
		if idx := strings.LastIndex(localPath, "/"); idx >= 0 {
			name = localPath[idx+1:]
		}
	} else if srcURL != "" {
		// Derive filename from upstream URL, strip query params.
		if idx := strings.LastIndex(srcURL, "/"); idx >= 0 && idx < len(srcURL)-1 {
			raw := srcURL[idx+1:]
			if qi := strings.IndexByte(raw, '?'); qi >= 0 {
				raw = raw[:qi]
			}
			if raw != "" {
				name = raw
			}
		}
	}
	return fmt.Sprintf("%s/plugins/packages/%s/%s", strings.TrimRight(base, "/"), checksum, name)
}
