package handler

import (
	"strings"

	"github.com/valyala/fasthttp"
)

var (
	Version   = "v1.0.0"
	GitCommit = "unknown"
)

func Router(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())

	switch {
	case path == "/health":
		Health(ctx)

	case path == "/manifest":
		handleUnifiedManifest(ctx)

	case strings.HasPrefix(path, "/api/"):
		APIRouter(ctx)

	case strings.HasPrefix(path, "/plugins/"):
		PluginRouter(ctx)

	default:
		ServeUI(ctx)
	}
}
