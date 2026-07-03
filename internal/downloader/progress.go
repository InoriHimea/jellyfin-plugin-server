package downloader

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// progressEntry tracks a single in-flight download. Done is updated by the
// copy loop via progressWriter; readers take atomic snapshots.
type progressEntry struct {
	VersionID string
	Checksum  string
	Filename  string
	Total     int64 // -1 when upstream sends no Content-Length
	StartedAt time.Time
	done      atomic.Int64
}

var progress sync.Map // versionID → *progressEntry

type progressWriter struct{ e *progressEntry }

func (w progressWriter) Write(p []byte) (int, error) {
	w.e.done.Add(int64(len(p)))
	return len(p), nil
}

// ActiveDownload is a point-in-time snapshot of one running download.
type ActiveDownload struct {
	VersionID  string  `json:"version_id"`
	Checksum   string  `json:"checksum"`
	Filename   string  `json:"filename"`
	DoneBytes  int64   `json:"done_bytes"`
	TotalBytes int64   `json:"total_bytes"`
	Percent    float64 `json:"percent"`
	SpeedBPS   int64   `json:"speed_bps"`
	ElapsedSec int64   `json:"elapsed_sec"`
}

// ActiveDownloads returns snapshots of all in-flight downloads,
// oldest first so the UI list stays stable across polls.
func ActiveDownloads() []ActiveDownload {
	out := []ActiveDownload{}
	progress.Range(func(_, v any) bool {
		e := v.(*progressEntry)
		done := e.done.Load()
		elapsed := time.Since(e.StartedAt)

		var speed int64
		if s := elapsed.Seconds(); s >= 0.5 {
			speed = int64(float64(done) / s)
		}
		pct := 0.0
		if e.Total > 0 {
			pct = float64(done) / float64(e.Total) * 100
			if pct > 100 {
				pct = 100
			}
		}
		out = append(out, ActiveDownload{
			VersionID:  e.VersionID,
			Checksum:   e.Checksum,
			Filename:   e.Filename,
			DoneBytes:  done,
			TotalBytes: e.Total,
			Percent:    pct,
			SpeedBPS:   speed,
			ElapsedSec: int64(elapsed.Seconds()),
		})
		return true
	})
	sort.Slice(out, func(i, j int) bool { return out[i].ElapsedSec > out[j].ElapsedSec })
	return out
}
