package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
	"golang.org/x/net/proxy"
)

const (
	defaultTimeout    = 60 * time.Second
	defaultMaxRetries = 3
	defaultRetryDelay = 2 * time.Second
)

type Response struct {
	StatusCode int
	Body       []byte
	ETag       string
	LastMod    string
}

var std *http.Client

func Init() {
	std = buildClient()
}

func buildClient() *http.Client {
	cfg := config.Get().Proxy
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          50,
	}

	switch cfg.Type {
	case config.ProxyHTTP, config.ProxyHTTPS:
		if cfg.Address != "" {
			proxyURLStr := string(cfg.Type) + "://"
			if cfg.Username != "" {
				proxyURLStr += url.UserPassword(cfg.Username, cfg.Password).String() + "@"
			}
			proxyURLStr += cfg.Address
			if pu, err := url.Parse(proxyURLStr); err == nil {
				transport.Proxy = http.ProxyURL(pu)
			}
		}
	case config.ProxySOCKS5:
		if cfg.Address != "" {
			var auth *proxy.Auth
			if cfg.Username != "" {
				auth = &proxy.Auth{User: cfg.Username, Password: cfg.Password}
			}
			if dialer, err := proxy.SOCKS5("tcp", cfg.Address, auth, proxy.Direct); err == nil {
				transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
					return dialer.Dial(network, addr)
				}
			}
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   defaultTimeout,
	}
}

// Get fetches a URL, optionally sending conditional headers.
func Get(rawURL, etag, lastMod string) (*Response, error) {
	if std == nil {
		Init()
	}

	var (
		resp    *http.Response
		lastErr error
	)

	for attempt := 0; attempt < defaultMaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(defaultRetryDelay)
		}

		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
		if lastMod != "" {
			req.Header.Set("If-Modified-Since", lastMod)
		}
		req.Header.Set("User-Agent", "jellyfin-plugin-server/1.0")

		resp, err = std.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		lastErr = nil
		break
	}
	if lastErr != nil {
		return nil, fmt.Errorf("fetch %s: %w", rawURL, lastErr)
	}
	defer resp.Body.Close()

	r := &Response{
		StatusCode: resp.StatusCode,
		ETag:       resp.Header.Get("ETag"),
		LastMod:    resp.Header.Get("Last-Modified"),
	}

	if resp.StatusCode != http.StatusNotModified {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read body: %w", err)
		}
		r.Body = body
	}
	return r, nil
}

// Reload rebuilds the client (called when proxy config changes).
func Reload() {
	std = buildClient()
}
