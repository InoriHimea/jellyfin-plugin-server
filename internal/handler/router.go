package handler

import (
	"strings"

	"github.com/valyala/fasthttp"
)

var (
	Version   = "v1.2.1"
	GitCommit = "unknown"
)

func Router(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())

	// Public endpoints — no auth required.
	switch {
	case path == "/health":
		Health(ctx); return
	case path == "/manifest":
		handleUnifiedManifest(ctx); return
	case strings.HasPrefix(path, "/plugins/"):
		PluginRouter(ctx); return
	case path == "/api/auth/login":
		apiLogin(ctx); return
	case path == "/api/auth/status":
		apiAuthStatus(ctx); return
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
