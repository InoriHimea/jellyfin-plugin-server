package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
	"github.com/inorihimea/jellyfin-plugin-server/internal/db"
	"github.com/inorihimea/jellyfin-plugin-server/internal/downloader"
	"github.com/inorihimea/jellyfin-plugin-server/internal/logger"
	"github.com/inorihimea/jellyfin-plugin-server/internal/manifest"
	proxyClient "github.com/inorihimea/jellyfin-plugin-server/internal/proxy"
	"github.com/valyala/fasthttp"
)

// PluginRouter handles:
//
//	GET /plugins/manifest/{repo-id}
//	GET /plugins/packages/{checksum}/{filename}
func PluginRouter(ctx *fasthttp.RequestCtx) {
	path := strings.TrimPrefix(string(ctx.Path()), "/plugins/")
	parts := strings.SplitN(path, "/", 3)

	switch {
	case len(parts) >= 2 && parts[0] == "manifest":
		handleManifest(ctx, parts[1])
	case len(parts) >= 3 && parts[0] == "packages":
		handlePackage(ctx, parts[1], parts[2])
	default:
		writeJSON(ctx, fasthttp.StatusNotFound, map[string]string{"error": "not found"})
	}
}

// handleUnifiedManifest serves GET /manifest — all enabled repos merged and deduplicated by GUID.
func handleUnifiedManifest(ctx *fasthttp.RequestCtx) {
	baseURL := baseURLFromCtx(ctx)
	catalog, err := manifest.BuildUnifiedManifest(baseURL)
	if err != nil {
		logger.Error("build unified manifest failed", map[string]any{"err": err})
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	b, _ := json.Marshal(catalog)
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(b)
}

func handleManifest(ctx *fasthttp.RequestCtx, repoID string) {
	repo, err := db.GetRepo(repoID)
	if err != nil {
		writeJSON(ctx, fasthttp.StatusNotFound, map[string]string{"error": "repo not found"})
		return
	}
	if !repo.Enabled {
		writeJSON(ctx, fasthttp.StatusForbidden, map[string]string{"error": "repo disabled"})
		return
	}

	cfg := config.Get()
	if manifest.IsTTLExpired(repo.LastFetched, cfg.Cache.ManifestTTLSeconds) {
		logger.Info("manifest TTL expired, refreshing", map[string]any{"repo": repo.Name})
		if _, _, err := manifest.FetchAndStore(repo.ID, repo.URL); err != nil {
			logger.Warn("upstream fetch failed, serving stale", map[string]any{"err": err, "repo": repo.Name})
		}
		db.RecordCacheAccess(false)
	} else {
		db.RecordCacheAccess(true)
	}

	baseURL := baseURLFromCtx(ctx)
	catalog, err := manifest.BuildLocalManifest(repo.ID, baseURL)
	if err != nil {
		logger.Error("build local manifest failed", map[string]any{"err": err})
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	b, _ := json.Marshal(catalog)
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(b)
}

// handlePackage serves plugin zip files.
// If the file is cached locally it is served from disk.
// Otherwise it is streamed from the upstream URL through our configured proxy,
// and a background download is triggered so subsequent requests are served locally.
func handlePackage(ctx *fasthttp.RequestCtx, checksum, filename string) {
	var versionID, localPath, sourceURL, dlStatus string
	err := db.DB.QueryRow(
		`SELECT id, COALESCE(local_path,''), source_url, download_status
		 FROM plugin_versions WHERE checksum=?`, checksum,
	).Scan(&versionID, &localPath, &sourceURL, &dlStatus)

	if err != nil {
		writeJSON(ctx, fasthttp.StatusNotFound, map[string]string{"error": "package not found"})
		return
	}

	// Fast path: already cached on disk.
	if dlStatus == "done" && localPath != "" {
		fasthttp.ServeFile(ctx, localPath)
		return
	}

	// Slow path: stream from upstream via our configured proxy.
	// Simultaneously enqueue a background download so it lands on disk for next time.
	downloader.Enqueue(versionID, checksum, sourceURL, filename)

	req, err := http.NewRequest(http.MethodGet, sourceURL, nil)
	if err != nil {
		logger.Error("build upstream request failed", map[string]any{"err": err, "url": sourceURL})
		ctx.Redirect(sourceURL, fasthttp.StatusFound)
		return
	}
	req.Header.Set("User-Agent", "jellyfin-plugin-server/1.0")

	// Use stream client (no body-read timeout) so large files don't get cut off.
	resp, err := proxyClient.GetStreamClient().Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		logger.Warn("upstream stream failed, redirecting", map[string]any{"err": err, "url": sourceURL})
		ctx.Redirect(sourceURL, fasthttp.StatusFound)
		return
	}
	defer resp.Body.Close()

	ctx.SetContentType("application/zip")
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		ctx.Response.Header.Set("Content-Length", cl)
	}
	ctx.Response.Header.Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	if _, err := io.Copy(ctx.Response.BodyWriter(), resp.Body); err != nil {
		logger.Warn("stream copy interrupted", map[string]any{"err": err})
	}
}

func baseURLFromCtx(ctx *fasthttp.RequestCtx) string {
	cfg := config.Get()

	// 1. Explicit override wins — set this when behind a reverse proxy.
	if cfg.Server.PublicURL != "" {
		return strings.TrimRight(cfg.Server.PublicURL, "/")
	}

	// 2. Respect reverse-proxy headers (X-Forwarded-Proto / X-Forwarded-Host).
	scheme := "http"
	if p := string(ctx.Request.Header.Peek("X-Forwarded-Proto")); p != "" {
		scheme = p
	} else if ctx.IsTLS() {
		scheme = "https"
	}

	host := string(ctx.Request.Header.Peek("X-Forwarded-Host"))
	if host == "" {
		host = string(ctx.Host())
	}
	if host == "" {
		host = fmt.Sprintf("localhost:%d", cfg.Server.Port)
	}

	return fmt.Sprintf("%s://%s", scheme, host)
}
