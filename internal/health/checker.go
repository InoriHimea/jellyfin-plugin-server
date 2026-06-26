package health

import (
	"sync"
	"time"

	"github.com/inorihimea/jellyfin-plugin-server/internal/db"
	"github.com/inorihimea/jellyfin-plugin-server/internal/logger"
	proxyClient "github.com/inorihimea/jellyfin-plugin-server/internal/proxy"
)

var (
	mu      sync.RWMutex
	results = map[string]RepoHealth{} // keyed by repo ID
)

// RepoHealth represents the last known health of an upstream repo.
type RepoHealth struct {
	RepoID    string    `json:"repo_id"`
	Name      string    `json:"name"`
	Reachable bool      `json:"reachable"`
	CheckedAt time.Time `json:"checked_at"`
	Error     string    `json:"error,omitempty"`
}

// Results returns a snapshot of all upstream health results.
func Results() []RepoHealth {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]RepoHealth, 0, len(results))
	for _, v := range results {
		out = append(out, v)
	}
	return out
}

// AllHealthy returns true if all enabled repos are reachable.
func AllHealthy() bool {
	mu.RLock()
	defer mu.RUnlock()
	for _, r := range results {
		if !r.Reachable {
			return false
		}
	}
	return true
}

// StartChecker begins a background goroutine that pings all enabled repos
// every checkInterval. It runs until the process exits.
func StartChecker(checkInterval time.Duration) {
	go func() {
		// Initial check shortly after startup.
		time.Sleep(10 * time.Second)
		runChecks()
		for range time.Tick(checkInterval) {
			runChecks()
		}
	}()
}

func runChecks() {
	repos, err := db.ListRepos()
	if err != nil {
		return
	}
	for _, r := range repos {
		if !r.Enabled {
			continue
		}
		rh := RepoHealth{RepoID: r.ID, Name: r.Name, CheckedAt: time.Now()}
		resp, err := proxyClient.Get(r.URL, "", "")
		if err != nil {
			rh.Reachable = false
			rh.Error = err.Error()
			logger.Warn("upstream unreachable", map[string]any{"repo": r.Name, "err": err})
			db.WriteLog("WARN", "upstream unreachable", "repo="+r.Name+" err="+err.Error())
		} else {
			rh.Reachable = resp.StatusCode < 500
			if !rh.Reachable {
				rh.Error = "upstream returned non-OK status"
				db.WriteLog("WARN", "upstream unhealthy", "repo="+r.Name)
			}
		}

		mu.Lock()
		results[r.ID] = rh
		mu.Unlock()
	}
}
