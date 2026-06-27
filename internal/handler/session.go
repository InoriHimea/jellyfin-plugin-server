package handler

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

const sessionTTL = 24 * time.Hour

var sessions = struct {
	mu    sync.RWMutex
	store map[string]time.Time
}{store: make(map[string]time.Time)}

func createSession() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	sessions.mu.Lock()
	sessions.store[token] = time.Now().Add(sessionTTL)
	sessions.mu.Unlock()
	return token, nil
}

func validateSession(token string) bool {
	if token == "" {
		return false
	}
	sessions.mu.RLock()
	exp, ok := sessions.store[token]
	sessions.mu.RUnlock()
	return ok && time.Now().Before(exp)
}

func revokeSession(token string) {
	sessions.mu.Lock()
	delete(sessions.store, token)
	sessions.mu.Unlock()
}

func tokenFromCtx(ctx *fasthttp.RequestCtx) string {
	auth := string(ctx.Request.Header.Peek("Authorization"))
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
