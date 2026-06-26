package db

import "time"

// RecordCacheAccess increments the hit or miss counter for the current hour.
func RecordCacheAccess(hit bool) {
	if DB == nil {
		return
	}
	hour := time.Now().UTC().Format("2006-01-02T15")
	col := "hits"
	if !hit {
		col = "misses"
	}
	DB.Exec(
		`INSERT INTO cache_stats (hour, hits, misses) VALUES (?, 0, 0)
		 ON CONFLICT(hour) DO NOTHING`, hour,
	)
	DB.Exec(
		`UPDATE cache_stats SET `+col+` = `+col+` + 1 WHERE hour = ?`, hour,
	)
}

// CacheHitRate returns the aggregate hit rate from the last N hours.
func CacheHitRate(hours int) (rate float64, totalHits, totalMisses int64) {
	if DB == nil {
		return 0, 0, 0
	}
	cutoff := time.Now().UTC().Add(-time.Duration(hours) * time.Hour).Format("2006-01-02T15")
	row := DB.QueryRow(
		`SELECT COALESCE(SUM(hits),0), COALESCE(SUM(misses),0)
		 FROM cache_stats WHERE hour >= ?`, cutoff,
	)
	row.Scan(&totalHits, &totalMisses)
	total := totalHits + totalMisses
	if total > 0 {
		rate = float64(totalHits) / float64(total)
	}
	return rate, totalHits, totalMisses
}
