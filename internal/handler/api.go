package handler

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
	"github.com/inorihimea/jellyfin-plugin-server/internal/db"
	"github.com/inorihimea/jellyfin-plugin-server/internal/downloader"
	"github.com/inorihimea/jellyfin-plugin-server/internal/logger"
	"github.com/inorihimea/jellyfin-plugin-server/internal/manifest"
	proxyClient "github.com/inorihimea/jellyfin-plugin-server/internal/proxy"
	"github.com/inorihimea/jellyfin-plugin-server/internal/storage"
	"github.com/valyala/fasthttp"
)

func APIRouter(ctx *fasthttp.RequestCtx) {
	path := strings.TrimPrefix(string(ctx.Path()), "/api")
	method := string(ctx.Method())

	switch {
	case path == "/status" && method == "GET":
		apiStatus(ctx)

	case path == "/config" && method == "GET":
		apiGetConfig(ctx)
	case path == "/config" && method == "PUT":
		apiPutConfig(ctx)

	case path == "/repos" && method == "GET":
		apiListRepos(ctx)
	case path == "/repos" && method == "POST":
		apiCreateRepo(ctx)

	case repoIDPath(path) && method == "PUT":
		apiUpdateRepo(ctx, repoID(path))
	case repoIDPath(path) && method == "DELETE":
		apiDeleteRepo(ctx, repoID(path))

	case repoActionPath(path, "refresh") && method == "POST":
		apiRefreshRepo(ctx, repoIDFromAction(path))
	case repoActionPath(path, "test") && method == "POST":
		apiTestRepo(ctx, repoIDFromAction(path))

	case path == "/repos/refresh-all" && method == "POST":
		apiRefreshAll(ctx)

	case path == "/logs" && method == "GET":
		apiLogs(ctx)

	case path == "/downloads/status" && method == "GET":
		apiDownloadsStatus(ctx)
	case path == "/downloads/retry-failed" && method == "POST":
		apiRetryFailed(ctx)

	case path == "/packages" && method == "GET":
		apiListPackages(ctx)
	case path == "/packages/cleanup" && method == "POST":
		apiCleanupPackages(ctx)
	case strings.HasPrefix(path, "/packages/") && method == "DELETE":
		apiDeletePackage(ctx, strings.TrimPrefix(path, "/packages/"))

	case path == "/auth/logout" && method == "POST":
		apiLogout(ctx)

	case path == "/catalog" && method == "GET":
		apiCatalog(ctx)
	case catalogDownloadPath(path) && method == "POST":
		apiCatalogDownload(ctx, catalogGUID(path))

	default:
		writeJSON(ctx, fasthttp.StatusNotFound, map[string]string{"error": "not found"})
	}
}

// ---- helpers for repo sub-paths ----

func repoIDPath(path string) bool {
	// /repos/{id}
	parts := strings.Split(strings.TrimPrefix(path, "/repos/"), "/")
	return strings.HasPrefix(path, "/repos/") && len(parts) == 1 && parts[0] != ""
}

func repoID(path string) string {
	return strings.TrimPrefix(path, "/repos/")
}

func repoActionPath(path, action string) bool {
	// /repos/{id}/{action}
	trimmed := strings.TrimPrefix(path, "/repos/")
	parts := strings.SplitN(trimmed, "/", 2)
	return len(parts) == 2 && parts[1] == action
}

func repoIDFromAction(path string) string {
	trimmed := strings.TrimPrefix(path, "/repos/")
	return strings.SplitN(trimmed, "/", 2)[0]
}

// ---- handlers ----

func apiStatus(ctx *fasthttp.RequestCtx) {
	type statusResp struct {
		Status     string `json:"status"`
		Version    string `json:"version"`
		GitCommit  string `json:"git_commit"`
		Uptime     string `json:"uptime"`
		DBOk       bool   `json:"db_ok"`
		DiskUsedMB int64  `json:"disk_used_mb"`
	}
	dbOk := db.DB != nil && db.DB.Ping() == nil
	diskBytes, _ := storage.DiskUsage()
	writeJSON(ctx, fasthttp.StatusOK, statusResp{
		Status:     "ok",
		Version:    Version,
		GitCommit:  GitCommit,
		Uptime:     time.Since(startTime).Round(time.Second).String(),
		DBOk:       dbOk,
		DiskUsedMB: diskBytes / 1024 / 1024,
	})
}

func apiListPackages(ctx *fasthttp.RequestCtx) {
	search := string(ctx.QueryArgs().Peek("q"))
	query := `
		SELECT v.id, p.name, p.owner, v.version, v.checksum, v.download_status,
		       COALESCE(v.local_path,''), v.source_url, COALESCE(v.downloaded_at,''),
		       COALESCE(v.fail_reason,'')
		FROM plugin_versions v JOIN plugins p ON p.id = v.plugin_id`
	args := []any{}
	if search != "" {
		query += ` WHERE p.name LIKE ? OR p.owner LIKE ?`
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	// No LIMIT: the frontend groups by plugin and paginates client-side, so it
	// expects the full result set. A hardcoded cap here silently truncated the
	// catalog (e.g. 200 rows out of 2000+) to whatever sorted first by name.
	query += ` ORDER BY p.name, v.version DESC`

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	type pkgEntry struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Owner        string `json:"owner"`
		Version      string `json:"version"`
		Checksum     string `json:"checksum"`
		Status       string `json:"status"`
		LocalPath    string `json:"local_path,omitempty"`
		SourceURL    string `json:"source_url"`
		DownloadedAt string `json:"downloaded_at,omitempty"`
		FailReason   string `json:"fail_reason,omitempty"`
	}
	var list []pkgEntry
	for rows.Next() {
		var e pkgEntry
		if err := rows.Scan(&e.ID, &e.Name, &e.Owner, &e.Version, &e.Checksum, &e.Status, &e.LocalPath, &e.SourceURL, &e.DownloadedAt, &e.FailReason); err != nil {
			continue
		}
		list = append(list, e)
	}
	writeJSON(ctx, fasthttp.StatusOK, list)
}

// apiDownloadsStatus returns aggregate download counters plus per-file
// progress snapshots for everything currently in flight.
func apiDownloadsStatus(ctx *fasthttp.RequestCtx) {
	summary := map[string]int{"pending": 0, "downloading": 0, "done": 0, "failed": 0, "total": 0}
	rows, err := db.DB.Query(
		`SELECT download_status, COUNT(*) FROM plugin_versions GROUP BY download_status`,
	)
	if err != nil {
		// Previously fell through silently and returned an all-zero summary,
		// which looks identical to "nothing queued" instead of "query failed"
		// (e.g. under SQLite write contention from a concurrent repo-refresh
		// burst) — logging and surfacing the error makes that distinguishable.
		logger.Warn("downloads status query failed", map[string]any{"err": err})
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var n int
		if rows.Scan(&status, &n) == nil {
			summary[status] = n
			summary["total"] += n
		}
	}

	type dlItem struct {
		downloader.ActiveDownload
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	active := downloader.ActiveDownloads()
	items := make([]dlItem, 0, len(active))
	for _, a := range active {
		item := dlItem{ActiveDownload: a}
		db.DB.QueryRow(
			`SELECT p.name, v.version FROM plugin_versions v
			 JOIN plugins p ON p.id = v.plugin_id WHERE v.id=?`, a.VersionID,
		).Scan(&item.Name, &item.Version)
		items = append(items, item)
	}

	writeJSON(ctx, fasthttp.StatusOK, map[string]any{
		"summary": summary,
		"active":  items,
	})
}

// apiRetryFailed re-enqueues every failed download.
func apiRetryFailed(ctx *fasthttp.RequestCtx) {
	// Flip failed → pending immediately so the UI shows a visible state
	// change right away, instead of waiting for a semaphore slot to free up
	// before anything appears to happen.
	res, err := db.DB.Exec(
		`UPDATE plugin_versions SET download_status='pending', fail_reason='' WHERE download_status='failed'`,
	)
	if err != nil {
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	go downloader.EnqueueAllPending()
	writeJSON(ctx, fasthttp.StatusOK, map[string]any{"retrying": n})
}

func apiDeletePackage(ctx *fasthttp.RequestCtx, checksum string) {
	var localPath, versionID string
	err := db.DB.QueryRow(
		`SELECT id, COALESCE(local_path,'') FROM plugin_versions WHERE checksum=?`, checksum,
	).Scan(&versionID, &localPath)
	if err != nil {
		writeJSON(ctx, fasthttp.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if localPath != "" {
		os.Remove(localPath)
	}
	db.DB.Exec(
		`UPDATE plugin_versions SET download_status='pending', local_path=NULL, downloaded_at=NULL WHERE id=?`,
		versionID,
	)
	writeJSON(ctx, fasthttp.StatusOK, map[string]string{"status": "ok"})
}

func apiGetConfig(ctx *fasthttp.RequestCtx) {
	writeJSON(ctx, fasthttp.StatusOK, config.Get())
}

func apiPutConfig(ctx *fasthttp.RequestCtx) {
	cfg := config.Defaults()
	if err := json.Unmarshal(ctx.PostBody(), cfg); err != nil {
		writeJSON(ctx, fasthttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := config.Update(cfg); err != nil {
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	proxyClient.Reload()
	writeJSON(ctx, fasthttp.StatusOK, map[string]string{"status": "ok"})
}

func apiListRepos(ctx *fasthttp.RequestCtx) {
	repos, err := db.ListRepos()
	if err != nil {
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, repos)
}

type repoInput struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

func apiCreateRepo(ctx *fasthttp.RequestCtx) {
	var in repoInput
	if err := json.Unmarshal(ctx.PostBody(), &in); err != nil {
		writeJSON(ctx, fasthttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if in.Name == "" || in.URL == "" {
		writeJSON(ctx, fasthttp.StatusBadRequest, map[string]string{"error": "name and url required"})
		return
	}
	repo, err := db.CreateRepo(in.Name, in.URL, in.Priority)
	if err != nil {
		if isUniqueErr(err) {
			writeJSON(ctx, fasthttp.StatusConflict, map[string]string{"error": "该 URL 已存在"})
			return
		}
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(ctx, fasthttp.StatusCreated, repo)
}

func apiUpdateRepo(ctx *fasthttp.RequestCtx, id string) {
	var in repoInput
	if err := json.Unmarshal(ctx.PostBody(), &in); err != nil {
		writeJSON(ctx, fasthttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := db.UpdateRepo(id, in.Name, in.URL, in.Enabled, in.Priority); err != nil {
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, map[string]string{"status": "ok"})
}

func apiDeleteRepo(ctx *fasthttp.RequestCtx, id string) {
	if err := db.DeleteRepo(id); err != nil {
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, map[string]string{"status": "ok"})
}

func apiRefreshRepo(ctx *fasthttp.RequestCtx, id string) {
	repo, err := db.GetRepo(id)
	if err != nil {
		writeJSON(ctx, fasthttp.StatusNotFound, map[string]string{"error": "repo not found"})
		return
	}
	// force re-fetch by clearing etag
	db.DB.Exec(`UPDATE repos SET etag='', last_fetched='' WHERE id=?`, id)
	_, changed, err := manifest.FetchAndStore(repo.ID, repo.URL)
	if err != nil {
		writeJSON(ctx, fasthttp.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, map[string]bool{"changed": changed})
}

func apiRefreshAll(ctx *fasthttp.RequestCtx) {
	repos, err := db.ListRepos()
	if err != nil {
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	type result struct {
		name string
		val  string
	}
	ch := make(chan result, len(repos))
	var wg sync.WaitGroup

	for _, r := range repos {
		if !r.Enabled {
			continue
		}
		wg.Add(1)
		go func(r db.Repo) {
			defer wg.Done()
			db.DB.Exec(`UPDATE repos SET etag='', last_fetched='' WHERE id=?`, r.ID)
			if _, _, err := manifest.FetchAndStore(r.ID, r.URL); err != nil {
				logger.Warn("refresh failed", map[string]any{"repo": r.Name, "err": err})
				ch <- result{r.Name, err.Error()}
			} else {
				ch <- result{r.Name, "ok"}
			}
		}(r)
	}

	wg.Wait()
	close(ch)

	results := make(map[string]string)
	for res := range ch {
		results[res.name] = res.val
	}
	writeJSON(ctx, fasthttp.StatusOK, results)
}

func apiTestRepo(ctx *fasthttp.RequestCtx, id string) {
	repo, err := db.GetRepo(id)
	if err != nil {
		writeJSON(ctx, fasthttp.StatusNotFound, map[string]string{"error": "repo not found"})
		return
	}
	resp, err := proxyClient.Get(repo.URL, "", "")
	if err != nil {
		writeJSON(ctx, fasthttp.StatusOK, map[string]any{"reachable": false, "error": err.Error()})
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, map[string]any{
		"reachable":   true,
		"status_code": resp.StatusCode,
	})
}

func apiLogs(ctx *fasthttp.RequestCtx) {
	search := string(ctx.QueryArgs().Peek("q"))
	logType := string(ctx.QueryArgs().Peek("type"))
	level := string(ctx.QueryArgs().Peek("level"))

	limit := ctx.QueryArgs().GetUintOrZero("limit")
	if limit <= 0 {
		limit = 50
	}
	if limit > 300 {
		limit = 300
	}
	offset := ctx.QueryArgs().GetUintOrZero("offset")

	var conditions []string
	args := []any{}
	if search != "" {
		conditions = append(conditions, `(message LIKE ? OR detail LIKE ?)`)
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	if logType != "" {
		conditions = append(conditions, `type = ?`)
		args = append(args, logType)
	}
	if level != "" {
		conditions = append(conditions, `level = ?`)
		args = append(args, level)
	}
	where := ""
	if len(conditions) > 0 {
		where = ` WHERE ` + strings.Join(conditions, " AND ")
	}

	// Total under the same filter, so the UI can render a real pager
	// instead of an unpageable most-recent-300 window.
	var total int64
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM logs`+where, args...).Scan(&total); err != nil {
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	rows, err := db.DB.Query(
		`SELECT id, type, level, message, COALESCE(detail,''), created_at FROM logs`+
			where+` ORDER BY id DESC LIMIT ? OFFSET ?`,
		append(append([]any{}, args...), limit, offset)...,
	)
	if err != nil {
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	type logEntry struct {
		ID        int64  `json:"id"`
		Type      string `json:"type"`
		Level     string `json:"level"`
		Message   string `json:"message"`
		Detail    string `json:"detail,omitempty"`
		CreatedAt string `json:"created_at"`
	}
	entries := []logEntry{}
	for rows.Next() {
		var e logEntry
		if err := rows.Scan(&e.ID, &e.Type, &e.Level, &e.Message, &e.Detail, &e.CreatedAt); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	writeJSON(ctx, fasthttp.StatusOK, map[string]any{"total": total, "entries": entries})
}

func writeJSON(ctx *fasthttp.RequestCtx, code int, v any) {
	b, _ := json.Marshal(v)
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(code)
	ctx.SetBody(b)
}

func apiCleanupPackages(ctx *fasthttp.RequestCtx) {
	dryRun := string(ctx.QueryArgs().Peek("dry_run")) == "1" ||
		string(ctx.QueryArgs().Peek("dry_run")) == "true"
	result, err := storage.RunCleanup(dryRun)
	if err != nil {
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, result)
}

func isUniqueErr(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// ---- catalog ----

func catalogDownloadPath(path string) bool {
	trimmed := strings.TrimPrefix(path, "/catalog/")
	return strings.HasPrefix(path, "/catalog/") && strings.HasSuffix(trimmed, "/download") && trimmed != "/download"
}

func catalogGUID(path string) string {
	trimmed := strings.TrimPrefix(path, "/catalog/")
	return strings.TrimSuffix(trimmed, "/download")
}

type catalogEntry struct {
	GUID          string `json:"guid"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Overview      string `json:"overview"`
	Owner         string `json:"owner"`
	Category      string `json:"category"`
	RepoName      string `json:"repo_name"`
	ImageURL      string `json:"image_url,omitempty"`
	VersionID     string `json:"version_id"`
	LatestVersion string `json:"latest_version"`
	LatestStatus  string `json:"latest_status"`
	VersionCount  int    `json:"version_count"`
}

func apiCatalog(ctx *fasthttp.RequestCtx) {
	rows, err := db.DB.Query(`
		SELECT p.guid, p.name, COALESCE(p.description,''), COALESCE(p.overview,''),
		       COALESCE(p.owner,''), COALESCE(p.category,''), r.name, COALESCE(p.image_url,''),
		       COALESCE((SELECT pv.id FROM plugin_versions pv WHERE pv.plugin_id=p.id ORDER BY pv.timestamp DESC LIMIT 1),''),
		       COALESCE((SELECT pv.version FROM plugin_versions pv WHERE pv.plugin_id=p.id ORDER BY pv.timestamp DESC LIMIT 1),''),
		       COALESCE((SELECT pv.download_status FROM plugin_versions pv WHERE pv.plugin_id=p.id ORDER BY pv.timestamp DESC LIMIT 1),''),
		       (SELECT COUNT(*) FROM plugin_versions WHERE plugin_id=p.id)
		FROM plugins p
		JOIN repos r ON r.id=p.repo_id
		WHERE r.enabled=1
		ORDER BY r.priority DESC, p.guid
	`)
	if err != nil {
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	seen := make(map[string]bool)
	var entries []catalogEntry
	for rows.Next() {
		var e catalogEntry
		if err := rows.Scan(&e.GUID, &e.Name, &e.Description, &e.Overview,
			&e.Owner, &e.Category, &e.RepoName, &e.ImageURL,
			&e.VersionID, &e.LatestVersion, &e.LatestStatus, &e.VersionCount); err != nil {
			continue
		}
		if seen[e.GUID] {
			continue
		}
		seen[e.GUID] = true
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []catalogEntry{}
	}
	writeJSON(ctx, fasthttp.StatusOK, entries)
}

func apiCatalogDownload(ctx *fasthttp.RequestCtx, guid string) {
	var versionID, checksum, sourceURL string
	err := db.DB.QueryRow(`
		SELECT pv.id, pv.checksum, pv.source_url
		FROM plugin_versions pv
		JOIN plugins p ON p.id=pv.plugin_id
		JOIN repos r ON r.id=p.repo_id
		WHERE p.guid=? AND r.enabled=1
		ORDER BY r.priority DESC, pv.timestamp DESC LIMIT 1
	`, guid).Scan(&versionID, &checksum, &sourceURL)
	if err != nil {
		writeJSON(ctx, fasthttp.StatusNotFound, map[string]string{"error": "plugin not found"})
		return
	}
	// Reset failed so it can be retried
	db.DB.Exec(`UPDATE plugin_versions SET download_status='pending' WHERE id=? AND download_status='failed'`, versionID)
	idx := strings.LastIndex(sourceURL, "/")
	filename := checksum + ".zip"
	if idx >= 0 && idx < len(sourceURL)-1 {
		filename = sourceURL[idx+1:]
	}
	downloader.Enqueue(versionID, checksum, sourceURL, filename)
	writeJSON(ctx, fasthttp.StatusOK, map[string]string{"status": "queued"})
}
