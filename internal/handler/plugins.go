package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
//	GET /plugins/images/{guid}
func PluginRouter(ctx *fasthttp.RequestCtx) {
	path := strings.TrimPrefix(string(ctx.Path()), "/plugins/")
	parts := strings.SplitN(path, "/", 3)

	switch {
	case len(parts) >= 2 && parts[0] == "manifest":
		handleManifest(ctx, parts[1])
	case len(parts) >= 3 && parts[0] == "packages":
		handlePackage(ctx, parts[1], parts[2])
	case len(parts) >= 2 && parts[0] == "images":
		handleImage(ctx, parts[1])
	default:
		writeJSON(ctx, fasthttp.StatusNotFound, map[string]string{"error": "not found"})
	}
}

// handleImage proxies and disk-caches plugin images so clients never fetch
// them from upstream hosts directly (which may be slow or unreachable).
func handleImage(ctx *fasthttp.RequestCtx, guid string) {
	// GUIDs come from our own manifest — reject anything path-like.
	if guid == "" || strings.ContainsAny(guid, "/\\.") {
		writeJSON(ctx, fasthttp.StatusBadRequest, map[string]string{"error": "bad guid"})
		return
	}

	cachePath := filepath.Join(config.ImagesDir(), guid)
	if data, err := os.ReadFile(cachePath); err == nil && len(data) > 0 {
		serveImage(ctx, data)
		return
	}

	var upstream string
	err := db.DB.QueryRow(
		`SELECT image_url FROM plugins WHERE guid=? AND image_url<>'' LIMIT 1`, guid,
	).Scan(&upstream)
	if err != nil || upstream == "" {
		writeJSON(ctx, fasthttp.StatusNotFound, map[string]string{"error": "no image"})
		return
	}

	resp, err := proxyClient.GetManifest(upstream, "", "")
	if err != nil || resp.StatusCode != http.StatusOK || len(resp.Body) == 0 {
		logger.Warn("image fetch failed", map[string]any{"err": err, "url": upstream})
		writeJSON(ctx, fasthttp.StatusBadGateway, map[string]string{"error": "image fetch failed"})
		return
	}

	if err := os.MkdirAll(config.ImagesDir(), 0755); err == nil {
		_ = os.WriteFile(cachePath, resp.Body, 0644)
	}
	serveImage(ctx, resp.Body)
}

func serveImage(ctx *fasthttp.RequestCtx, data []byte) {
	ctx.SetContentType(http.DetectContentType(data))
	ctx.Response.Header.Set("Cache-Control", "public, max-age=604800")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(data)
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
		// Stale-while-revalidate: serve cached data immediately, refresh in background.
		// This prevents upstream latency from blocking Jellyfin's package catalog request.
		db.RecordCacheAccess(false)
		go func() {
			if _, _, err := manifest.FetchAndStore(repo.ID, repo.URL); err != nil {
				logger.Warn("background manifest refresh failed", map[string]any{"err": err, "repo": repo.Name})
			}
		}()
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
// If the file is cached locally it is served from disk. Otherwise it is
// downloaded synchronously (through the configured proxy, checksum-verified,
// deduplicated via singleflight) and then served — never a raw pass-through
// stream, so Jellyfin can't receive a truncated or corrupt body.
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
		if _, statErr := os.Stat(localPath); statErr == nil {
			serveZip(ctx, localPath, filename)
			return
		}
		// DB says done but the file is gone — fall through and re-download.
	}

	// Cache miss: download to disk now (checksum-verified), then serve the file.
	// Concurrent requests for the same checksum share one download via singleflight.
	if err := downloader.EnqueueSync(versionID, checksum, sourceURL, filename); err != nil {
		logger.Warn("on-demand download failed", map[string]any{"err": err, "url": sourceURL})
		writeJSON(ctx, fasthttp.StatusBadGateway, map[string]string{"error": "upstream download failed"})
		return
	}

	var freshPath string
	if err := db.DB.QueryRow(
		`SELECT COALESCE(local_path,'') FROM plugin_versions WHERE id=?`, versionID,
	).Scan(&freshPath); err != nil || freshPath == "" {
		writeJSON(ctx, fasthttp.StatusBadGateway, map[string]string{"error": "download completed but file missing"})
		return
	}
	serveZip(ctx, freshPath, filename)
}

func serveZip(ctx *fasthttp.RequestCtx, path, filename string) {
	ctx.Response.Header.Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	ctx.SetContentType("application/zip")
	fasthttp.ServeFile(ctx, path)
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
