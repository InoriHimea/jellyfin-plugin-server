package handler

import (
	"encoding/json"

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
	"github.com/valyala/fasthttp"
)

// apiAuthStatus returns whether auth is currently enabled.
// Public endpoint — no token required.
func apiAuthStatus(ctx *fasthttp.RequestCtx) {
	cfg := config.Get()
	writeJSON(ctx, fasthttp.StatusOK, map[string]bool{
		"enabled": cfg.Auth.Enabled && cfg.Auth.Username != "",
	})
}

// apiLogin validates credentials and issues a session token.
// Public endpoint — no token required.
func apiLogin(ctx *fasthttp.RequestCtx) {
	if string(ctx.Method()) != "POST" {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}

	cfg := config.Get()

	// Auth disabled: issue token without credential check.
	if !cfg.Auth.Enabled || cfg.Auth.Username == "" {
		token, err := createSession()
		if err != nil {
			writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": "could not create session"})
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, map[string]string{"token": token})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeJSON(ctx, fasthttp.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.Username != cfg.Auth.Username || req.Password != cfg.Auth.Password {
		writeJSON(ctx, fasthttp.StatusUnauthorized, map[string]string{"error": "用户名或密码错误"})
		return
	}

	token, err := createSession()
	if err != nil {
		writeJSON(ctx, fasthttp.StatusInternalServerError, map[string]string{"error": "could not create session"})
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, map[string]string{"token": token})
}

// apiLogout revokes the current session token.
// Protected — requires a valid token.
func apiLogout(ctx *fasthttp.RequestCtx) {
	revokeSession(tokenFromCtx(ctx))
	writeJSON(ctx, fasthttp.StatusOK, map[string]bool{"ok": true})
}
