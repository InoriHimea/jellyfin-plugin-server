package handler

import "testing"

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
