package storage

import (
	"os"
	"path/filepath"

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
)

// DiskUsage returns the total bytes used under the packages directory.
func DiskUsage() (int64, error) {
	dir := config.PackagesDir()
	var total int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}
