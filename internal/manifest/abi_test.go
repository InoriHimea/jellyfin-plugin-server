package manifest

import "testing"

func TestNormalizeTargetABIForManifest(t *testing.T) {
	tests := []struct {
		abi, app, want string
	}{
		{"12.0.0.0", "12.0.0", "12.0.0"},
		{"10.11.8.0", "10.11.8", "10.11.8"},
		{"13.0.0.0", "13.0.0", "13.0.0"},
		{"13.0.0.0", "12.0.0", "13.0.0.0"},
		{"12.0.0.1", "12.0.0", "12.0.0.1"},
		{"12.0.1.0", "12.0.0", "12.0.1.0"},
		{"12.0.0.0", "12.0.0.0", "12.0.0.0"},
		{"12.0.0", "12.0.0", "12.0.0"},
		{"12.0.0.00", "12.0.0", "12.0.0"},
		{"12.0.x.0", "12.0.0", "12.0.x.0"},
		{"12.0.0.0", "", "12.0.0.0"},
		{"12.0.0.0", "12.0", "12.0.0.0"},
		{"", "12.0.0", ""},
	}
	for _, tt := range tests {
		t.Run(tt.abi+"/"+tt.app, func(t *testing.T) {
			if got := normalizeTargetABIForManifest(tt.abi, tt.app); got != tt.want {
				t.Errorf("normalizeTargetABIForManifest() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeManifestVersionsDeduplicatesCanonicalABI(t *testing.T) {
	versions := []Version{
		{Version: "31.0.0.0", TargetABI: "12.0.0.0", Checksum: "raw-four-part"},
		{Version: "31.0.0.0", TargetABI: "12.0.0", Checksum: "raw-three-part"},
		{Version: "31.0.0.0", TargetABI: "13.0.0.0", Checksum: "abi13"},
	}
	got := normalizeManifestVersions(versions, "12.0.0")
	if len(got) != 2 {
		t.Fatalf("got %d versions, want 2", len(got))
	}
	if got[0].TargetABI != "12.0.0" || got[0].Checksum != "raw-four-part" {
		t.Errorf("canonical collision winner = %#v, want first raw record", got[0])
	}
	if versions[0].TargetABI != "12.0.0.0" || versions[1].TargetABI != "12.0.0" {
		t.Errorf("normalization mutated input ABI values: %#v", versions)
	}
}
