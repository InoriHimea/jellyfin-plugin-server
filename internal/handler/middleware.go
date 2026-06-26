package handler

import (
	"fmt"
	"runtime/debug"
	"time"

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
