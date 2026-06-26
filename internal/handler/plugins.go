package handler

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
	"github.com/inorihimea/jellyfin-plugin-server/internal/db"
	"github.com/inorihimea/jellyfin-plugin-server/internal/logger"
	"github.com/inorihimea/jellyfin-plugin-server/internal/manifest"
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
		db.RecordCacheAccess(false) // upstream fetch = miss
	} else {
		db.RecordCacheAccess(true) // TTL still valid = hit
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

func handlePackage(ctx *fasthttp.RequestCtx, checksum, filename string) {
	var localPath, sourceURL, dlStatus string
	err := db.DB.QueryRow(
		`SELECT COALESCE(local_path,''), source_url, download_status
		 FROM plugin_versions WHERE checksum=?`, checksum,
	).Scan(&localPath, &sourceURL, &dlStatus)

	if err != nil {
		writeJSON(ctx, fasthttp.StatusNotFound, map[string]string{"error": "package not found"})
		return
	}

	if dlStatus == "done" && localPath != "" {
		fasthttp.ServeFile(ctx, localPath)
		return
	}

	// Fallback: redirect to upstream
	ctx.Redirect(sourceURL, fasthttp.StatusFound)
}

func baseURLFromCtx(ctx *fasthttp.RequestCtx) string {
	scheme := "http"
	if ctx.IsTLS() {
		scheme = "https"
	}
	host := string(ctx.Host())
	if host == "" {
		cfg := config.Get()
		host = fmt.Sprintf("localhost:%d", cfg.Server.Port)
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}
