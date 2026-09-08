package manifest

import "testing"

func TestExtractJellyfinVersion(t *testing.T) {
	tests := []struct {
		ua   string
		want string
	}{
		{"Jellyfin/10.11.8.0", "10.11.8"},
		{"Jellyfin/10.10.7", "10.10.7"},
		{"Mozilla/5.0 Jellyfin/10.9.2.0 curl", "10.9.2"},
		{"MediaBrowser/10.8.13 (Jellyfin)", "10.8.13"},
		{"curl/8.0", ""},
		{"", ""},
		{"Jellyfin_server/10.10.0", ""}, // wrong product name: no slash-segment match
		{"jellyfin/10.10.0", ""},        // UA product names are case-sensitive
	}
	for _, tt := range tests {
		if got := extractJellyfinVersion(tt.ua); got != tt.want {
			t.Errorf("extractJellyfinVersion(%q) = %q, want %q", tt.ua, got, tt.want)
		}
	}
}
