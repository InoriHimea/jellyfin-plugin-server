package pkgcheck

import (
	"archive/zip"
	"bytes"
	"testing"
)

func buildZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func TestScanZipDotnetMajor(t *testing.T) {
	dll8 := []byte("garbage\x00\x01MIME\x00.NETCoreApp,Version=v8.0\x00more garbage")
	dll10 := []byte("\x00.NETCoreApp,Version=v10.0\x00")
	legacy := []byte(".NETFramework,Version=v4.7.2 no coreapp here")

	tests := []struct {
		name string
		zip  map[string][]byte
		want int
	}{
		{"net8 plugin", map[string][]byte{"Webhook.dll": dll8}, 8},
		{"net10 dependency wins", map[string][]byte{"Webhook.dll": dll8, "MimeKit.dll": dll10}, 10},
		{"legacy framework ignored", map[string][]byte{"Plugin.dll": legacy}, 0},
		{"no dlls", map[string][]byte{"README.md": []byte("hello")}, 0},
		{"corrupt zip", nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte("this is not a zip")
			if tt.zip != nil {
				data = buildZip(t, tt.zip)
			}
			if got := ScanZipDotnetMajor(data); got != tt.want {
				t.Errorf("ScanZipDotnetMajor() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMaxDotnetMajor(t *testing.T) {
	tests := []struct {
		version string
		want    int
	}{
		{"10.8.13.0", 6},
		{"10.9.2.0", 8},
		{"10.10.7.0", 8},
		{"10.11.8.0", 9},
		{"10.12.0.0", 9},  // unmapped future series → last known table entry
		{"11.0.0.0", 0},   // future major: unmapped → refuse to filter
		{"9.10.10.0", 0},  // pre-10 Jellyfin → refuse to filter
		{"", 0},
		{"garbage", 0},
		{"10.11", 9},
	}
	for _, tt := range tests {
		if got := MaxDotnetMajor(tt.version); got != tt.want {
			t.Errorf("MaxDotnetMajor(%q) = %d, want %d", tt.version, got, tt.want)
		}
	}
}
