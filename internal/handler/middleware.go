package handler

import (
	"encoding/base64"
	"fmt"
	"runtime/debug"
	"strings"
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

// authOK checks Basic Auth when auth is enabled. Returns true if the request
// may proceed. On failure it writes 401 and returns false.
// Public paths (/manifest, /health, /plugins/*) bypass this — call it only
// for protected routes.
func authOK(ctx *fasthttp.RequestCtx) bool {
	cfg := config.Get()
	if !cfg.Auth.Enabled || cfg.Auth.Username == "" {
		return true
	}
	auth := string(ctx.Request.Header.Peek("Authorization"))
	user, pass, ok := parseBasicAuth(auth)
	if !ok || user != cfg.Auth.Username || pass != cfg.Auth.Password {
		ctx.Response.Header.Set("WWW-Authenticate", `Basic realm="Jellyfin Plugin Server"`)
		ctx.SetContentType("application/json")
		ctx.SetStatusCode(fasthttp.StatusUnauthorized)
		ctx.SetBodyString(`{"error":"unauthorized"}`)
		return false
	}
	return true
}

func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return "", "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, prefix))
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func withPanic(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered", map[string]any{
					"panic": fmt.Sprintf("%v", r),
					"stack": string(debug.Stack()),
				})
				ctx.SetContentType("application/json")
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
				ctx.SetBodyString(`{"error":"internal server error"}`)
			}
		}()
		h(ctx)
	}
}
