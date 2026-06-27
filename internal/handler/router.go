package handler

import (
	"strings"

	"github.com/valyala/fasthttp"
)

var (
	Version   = "v1.3.1"
	GitCommit = "unknown"
)

func Router(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())

	// Fully public endpoints (no auth, no SPA).
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

	// All non-API paths → serve the React SPA.
	// The SPA handles /login and auth-gating itself; no server-side check here.
	if !strings.HasPrefix(path, "/api/") {
		ServeUI(ctx)
		return
	}

	// API routes require a valid session token.
	if !authOK(ctx) {
		return
	}
	APIRouter(ctx)
}
