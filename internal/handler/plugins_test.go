package handler

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestResolvePackageJellyfinVersion(t *testing.T) {
	tests := []struct {
		name           string
		uri            string
		userAgent      string
		defaultVersion string
		want           string
		wantErr        bool
	}{
		{"jv wins over conflicting UA", "/plugins/packages/a/file.zip?jv=12.0.0", "Jellyfin/10.11.8.0", "10.11.8", "12.0.0", false},
		{"jv normalizes prefix", "/plugins/packages/a/file.zip?jv=v12.0.0", "", "10.11.8", "12.0.0", false},
		{"UA fallback", "/plugins/packages/a/file.zip", "Jellyfin/12.0.0.0", "10.11.8", "12.0.0", false},
		{"configured fallback", "/plugins/packages/a/file.zip", "curl/8.0", "10.11.8", "10.11.8", false},
		{"invalid jv rejected", "/plugins/packages/a/file.zip?jv=not-a-version", "Jellyfin/12.0.0.0", "10.11.8", "", true},
		{"duplicate jv rejected", "/plugins/packages/a/file.zip?jv=12.0.0&jv=10.11.8", "", "10.11.8", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx fasthttp.RequestCtx
			ctx.Request.SetRequestURI(tt.uri)
			ctx.Request.Header.SetUserAgent(tt.userAgent)
			got, err := resolvePackageJellyfinVersion(&ctx, tt.defaultVersion)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr=%v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("version = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveJellyfinVersion(t *testing.T) {
	tests := []struct {
		name              string
		pathVersion       string
		userAgent         string
		configuredDefault string
		want              string
	}{
		{"path wins over UA and config", "10.11.8", "Jellyfin/12.0.0.0", "10.10.7", "10.11.8"},
		{"path strips v prefix", "v10.11.8", "Jellyfin/12.0.0.0", "10.10.7", "10.11.8"},
		{"UA wins over config", "", "Jellyfin/12.0.0.0", "10.11.8", "12.0.0"},
		{"config is fallback", "", "curl/8.0", "10.11.8", "10.11.8"},
		{"empty inputs do not filter", "", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveJellyfinVersion(tt.pathVersion, tt.userAgent, tt.configuredDefault); got != tt.want {
				t.Errorf("resolveJellyfinVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
