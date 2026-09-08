// Package pkgcheck inspects downloaded plugin zips for .NET runtime
// compatibility. Jellyfin loads plugin assemblies on its own .NET runtime,
// so a zip whose DLLs were compiled against a newer .NET than the server's
// runtime fails with NotSupported at load time — even though the manifest's
// targetAbi parses fine and the file's checksum matches its mirror. This
// package detects that case from the package content itself.
package pkgcheck

import (
	"archive/zip"
	"bytes"
	"io"
	"regexp"
)

// targetFrameworkMarker matches the TargetFrameworkAttribute string that the
// C#/F# compiler embeds in every assembly built against .NET (Core) —
// e.g. ".NETCoreApp,Version=v8.0" or ".NETFramework,Version=v4.7.2". The
// attribute lives uncompressed in the #Strings heap, so scanning the raw
// bytes is equivalent to parsing ECMA-335 metadata for this purpose.
var targetFrameworkMarker = regexp.MustCompile(`\.NETCoreApp,Version=v(\d+)\.\d+`)

// ScanZipDotnetMajor returns the highest .NET major version any DLL in the
// zip was compiled against, or 0 when nothing declares one (native-only
// zips, legacy .NET Framework builds, or no assemblies at all).
func ScanZipDotnetMajor(data []byte) int {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return 0
	}

	highest := 0
	for _, f := range zr.File {
		if !isAssemblyCandidate(f.Name) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		// #Strings is near the file start; cap the read — full DLLs can be
		// tens of MB and the marker never appears beyond the metadata heaps.
		buf := make([]byte, 4<<20)
		n, _ := io.ReadFull(rc, buf)
		rc.Close()

		for _, m := range targetFrameworkMarker.FindAllSubmatch(buf[:n], -1) {
			major := parseMajor(m[1])
			if major > highest {
				highest = major
			}
		}
	}
	return highest
}

func isAssemblyCandidate(name string) bool {
	return len(name) >= 4 && (name[len(name)-4:] == ".dll" || name[len(name)-4:] == ".exe")
}

func parseMajor(b []byte) int {
	n := 0
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
