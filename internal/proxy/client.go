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

	// Manifests are small JSON files — tight timeout so slow/dead repos
	// don't stall the parallel refresh of all 90+ repos.
	manifestTimeout    = 15 * time.Second
	manifestMaxRetries = 1
	manifestRetryDelay = 1 * time.Second
)

type Response struct {
	StatusCode int
	Body       []byte
	ETag       string
	LastMod    string
}

var std *http.Client
var mfClient *http.Client

func Init() {
	std = buildClient()
	mfClient = buildManifestClient()
}

func buildManifestClient() *http.Client {
	cfg := config.Get().Proxy
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   8 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   8 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       60 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   4,
		// Hard cap on concurrent connections per host. Without this, a
		// catalog page full of GitHub-hosted images (all the same host)
		// can fan out to hundreds of simultaneous dials, which on a
		// constrained NAS network path stalls everything else too.
		MaxConnsPerHost: 8,
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
	return &http.Client{Transport: transport, Timeout: manifestTimeout}
}

// GetManifest fetches a manifest JSON with a short timeout and single retry.
// Use this instead of Get for manifest URLs — manifests are tiny files and
// slow repos should not block the parallel startup/refresh of 90+ repos.
func GetManifest(rawURL, etag, lastMod string) (*Response, error) {
	if mfClient == nil {
		mfClient = buildManifestClient()
	}
	var (
		resp    *http.Response
		lastErr error
	)
	for attempt := 0; attempt <= manifestMaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(manifestRetryDelay)
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
		resp, err = mfClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		lastErr = nil
		break
	}
	if lastErr != nil {
		return nil, fmt.Errorf("fetch manifest %s: %w", rawURL, lastErr)
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
	streamClient = buildStreamClient()
	mfClient = buildManifestClient()
}

// GetClient returns the configured HTTP client for direct streaming use.
func GetClient() *http.Client {
	if std == nil {
		Init()
	}
	return std
}

// GetStreamClient returns a client with no overall body-read timeout, suitable
// for proxying large plugin zip files that may take minutes to download.
func GetStreamClient() *http.Client {
	if streamClient == nil {
		streamClient = buildStreamClient()
	}
	return streamClient
}

var streamClient *http.Client

func buildStreamClient() *http.Client {
	cfg := config.Get().Proxy
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          10,
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

	// No Timeout: body reads are unlimited so large files don't get cut off.
	return &http.Client{Transport: transport}
}
