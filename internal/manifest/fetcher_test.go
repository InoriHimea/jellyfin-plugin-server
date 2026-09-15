package manifest

import (
	"strings"
	"testing"
)

func TestLocalURLCarriesCompatibilityContext(t *testing.T) {
	got := localURL("https://plugins.example/", "abc", "", "https://upstream.example/Trakt.zip?token=x", "12.0.0")
	want := "https://plugins.example/plugins/packages/abc/Trakt.zip?jv=12.0.0"
	if got != want {
		t.Fatalf("localURL() = %q, want %q", got, want)
	}
	if strings.Contains(localURL("https://plugins.example", "abc", "", "https://upstream.example/Trakt.zip", ""), "?jv=") {
		t.Error("empty compatibility version must not add jv")
	}
}

func TestIsVersionCompatible(t *testing.T) {
	tests := []struct {
		name          string
		abi           string
		dotnetMajor   int
		targetVersion string
		want          bool
	}{
		{"Jellyfin 12 accepts ABI 12 .NET 10", "12.0.0", 10, "12.0.0", true},
		{"Jellyfin 12 rejects ABI 13", "13.0.0", 10, "12.0.0", false},
		{"Jellyfin 10.11 rejects ABI 12", "12.0.0", 10, "10.11.8", false},
		{"Jellyfin 12 rejects .NET 11", "12.0.0", 11, "12.0.0", false},
		{"unknown target preserves legacy behavior", "13.0.0", 11, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVersionCompatible(tt.abi, tt.dotnetMajor, tt.targetVersion); got != tt.want {
				t.Errorf("IsVersionCompatible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterIncompatibleVersions(t *testing.T) {
	versions := []Version{
		{Version: "11", TargetABI: "10.11.0", DotnetMajor: 9},
		{Version: "12", TargetABI: "12.0.0", DotnetMajor: 10},
		{Version: "net11", TargetABI: "12.0.0", DotnetMajor: 11},
		{Version: "unknown", TargetABI: "", DotnetMajor: 0},
	}

	tests := []struct {
		name      string
		target    string
		dotnetCap int
		want      []string
	}{
		{"Jellyfin 10.11 excludes ABI 12", "10.11.8", 9, []string{"11", "unknown"}},
		{"Jellyfin 12 includes .NET 10 but excludes .NET 11", "12.0.0", 10, []string{"11", "12", "unknown"}},
		{"empty target keeps all variants", "", 0, []string{"11", "12", "net11", "unknown"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterIncompatibleVersions(append([]Version(nil), versions...), tt.target, tt.dotnetCap)
			if len(got) != len(tt.want) {
				t.Fatalf("kept %d versions, want %d: %#v", len(got), len(tt.want), got)
			}
			for i, want := range tt.want {
				if got[i].Version != want {
					t.Errorf("version[%d] = %q, want %q", i, got[i].Version, want)
				}
			}
		})
	}
}
