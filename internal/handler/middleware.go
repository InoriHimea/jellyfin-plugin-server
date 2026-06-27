package handler

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
	"github.com/inorihimea/jellyfin-plugin-server/internal/logger"
	"github.com/valyala/fasthttp"
)

// Chain wraps a handler with panic recovery and request logging.
func Chain(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return withPanic(withLogger(h))
}

func withLogger(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		start := time.Now()
		h(ctx)
		logger.Info("request", map[string]any{
			"method":     string(ctx.Method()),
			"path":       string(ctx.Path()),
			"status":     ctx.Response.StatusCode(),
			"latency_ms": time.Since(start).Milliseconds(),
			"ip":         ctx.RemoteIP().String(),
		})
	}
}

// authOK validates a Bearer session token when auth is enabled.
// Returns true if the request may proceed.
// Call this only for protected routes — public paths bypass it.
func authOK(ctx *fasthttp.RequestCtx) bool {
	cfg := config.Get()
	if !cfg.Auth.Enabled || cfg.Auth.Username == "" {
		return true
	}
	if !validateSession(tokenFromCtx(ctx)) {
		writeJSON(ctx, fasthttp.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return false
	}
	return true
}

func withPanic(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered", map[string]any{
					"panic": fmt.Sprintf("%v", r),
					"stack": string(debug.Stack()),
				})
				writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
		}()
		h(ctx)
	}
}
