package handler

import (
	"strings"

	"github.com/valyala/fasthttp"
)

var (
	Version   = "v1.1.0"
	GitCommit = "unknown"
)

func Router(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())

	// Public endpoints — Jellyfin clients poll these without credentials.
	switch {
	case path == "/health":
		Health(ctx)
		return
	case path == "/manifest":
		handleUnifiedManifest(ctx)
		return
	case strings.HasPrefix(path, "/plugins/"):
		PluginRouter(ctx)
		return
	}

	// All remaining routes (API + web UI) require auth when enabled.
	if !authOK(ctx) {
		return
	}

	switch {
	case strings.HasPrefix(path, "/api/"):
		APIRouter(ctx)
	default:
		ServeUI(ctx)
	}
}
