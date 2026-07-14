package handler

import (
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
	"github.com/inorihimea/jellyfin-plugin-server/internal/db"
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
		path := string(ctx.Path())
		status := ctx.Response.StatusCode()
		latencyMs := time.Since(start).Milliseconds()
		ip := clientIP(ctx)

		logger.Info("request", map[string]any{
			"method":     string(ctx.Method()),
			"path":       path,
			"status":     status,
			"latency_ms": latencyMs,
			"ip":         ip,
		})

		// Persist an audit trail for the public, unauthenticated endpoints —
		// this server is exposed on the internet, and these are the paths
		// Jellyfin (or anyone else) can hit without logging in. Authenticated
		// /api/* admin traffic is deliberately excluded: it's already gated by
		// login (itself audited in apiLogin), and logging every 2-second
		// dashboard poll would bury anything worth seeing.
		if isAuditPath(path) {
			go db.WriteLogTyped("access", "INFO", "http access", fmt.Sprintf(
				"ip=%s method=%s path=%s status=%d latency_ms=%d",
				ip, string(ctx.Method()), path, status, latencyMs,
			))
		}
	}
}

func isAuditPath(path string) bool {
	// /health is excluded: it's hit by docker/uptime monitoring, not Jellyfin,
	// and would just drown the log in infra noise.
	return path == "/manifest" || strings.HasPrefix(path, "/plugins/")
}

// clientIP returns the real client address. This server is typically
// deployed behind a reverse proxy (baseURLFromCtx already trusts
// X-Forwarded-Host/-Proto for the same reason), so the raw TCP peer seen by
// fasthttp is usually just the proxy — X-Forwarded-For/X-Real-IP carry the
// actual origin when present.
func clientIP(ctx *fasthttp.RequestCtx) string {
	if xff := string(ctx.Request.Header.Peek("X-Forwarded-For")); xff != "" {
		if idx := strings.IndexByte(xff, ','); idx >= 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := string(ctx.Request.Header.Peek("X-Real-IP")); xri != "" {
		return strings.TrimSpace(xri)
	}
	return ctx.RemoteIP().String()
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
