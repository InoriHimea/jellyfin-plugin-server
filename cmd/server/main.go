package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
	"github.com/inorihimea/jellyfin-plugin-server/internal/db"
	"github.com/inorihimea/jellyfin-plugin-server/internal/downloader"
	"github.com/inorihimea/jellyfin-plugin-server/internal/handler"
	"github.com/inorihimea/jellyfin-plugin-server/internal/health"
	"github.com/inorihimea/jellyfin-plugin-server/internal/logger"
	"github.com/inorihimea/jellyfin-plugin-server/internal/manifest"
	"github.com/inorihimea/jellyfin-plugin-server/internal/storage"
	"github.com/valyala/fasthttp"
)

func main() {
	// Resolve config path inside the data dir so it survives container restarts.
	dataDir := "./data"
	if v := os.Getenv("JPSERVER_DATA_DIR"); v != "" {
		dataDir = v
	}
	cfgFile := filepath.Join(dataDir, "config.json")
	if v := os.Getenv("JPSERVER_CONFIG"); v != "" {
		cfgFile = v
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	logger.Init(cfg.LogJSON)
	logger.Info("starting jellyfin-plugin-server", map[string]any{"version": handler.Version})

	if err := db.Open(config.DBPath()); err != nil {
		logger.Error("db open failed", map[string]any{"err": err})
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database ready", map[string]any{"path": config.DBPath()})

	if err := db.SeedDefaultRepos(); err != nil {
		logger.Warn("seed repos failed", map[string]any{"err": err})
	}

	manifest.SetEnqueueFunc(downloader.EnqueueAllPending)
	go downloader.EnqueueAllPending()
	go scheduledCleanup()
	go startupRefresh(cfg) // warm the DB on startup so /manifest is immediately populated
	health.StartChecker(5 * time.Minute)

	if err := os.MkdirAll(config.PackagesDir(), 0755); err != nil {
		logger.Error("create packages dir failed", map[string]any{"err": err})
		os.Exit(1)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &fasthttp.Server{
		Handler: handler.Chain(handler.Router),
		Name:    "jellyfin-plugin-server/" + handler.Version,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("server listening", map[string]any{"addr": addr})
		if err := srv.ListenAndServe(addr); err != nil {
			logger.Error("server error", map[string]any{"err": err})
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("shutting down...")
	srv.Shutdown()
	logger.Info("waiting for in-flight downloads...")
	downloader.Wait()
	logger.Info("bye")
}

// startupRefresh fetches all enabled repos in parallel when the server boots.
// It only fetches repos whose TTL has expired, so subsequent restarts are cheap.
func startupRefresh(cfg *config.Config) {
	repos, err := db.ListRepos()
	if err != nil {
		logger.Warn("startup refresh: list repos failed", map[string]any{"err": err})
		return
	}
	var wg sync.WaitGroup
	for _, r := range repos {
		if !r.Enabled {
			continue
		}
		if !manifest.IsTTLExpired(r.LastFetched, cfg.Cache.ManifestTTLSeconds) {
			continue
		}
		wg.Add(1)
		go func(r db.Repo) {
			defer wg.Done()
			if _, _, err := manifest.FetchAndStore(r.ID, r.URL); err != nil {
				logger.Warn("startup refresh failed", map[string]any{"repo": r.Name, "err": err})
			} else {
				logger.Info("startup refresh ok", map[string]any{"repo": r.Name})
			}
		}(r)
	}
	wg.Wait()
	logger.Info("startup refresh complete", nil)
}

// scheduledCleanup runs storage cleanup once a day at 03:00 local time.
func scheduledCleanup() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, time.Local)
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		time.Sleep(time.Until(next))
		if result, err := storage.RunCleanup(false); err != nil {
			logger.Warn("scheduled cleanup error", map[string]any{"err": err})
		} else {
			logger.Info("scheduled cleanup done", map[string]any{
				"lru":        len(result.LRURemoved),
				"orphan":     len(result.OrphanRemoved),
				"bytes_freed": result.BytesFreed,
			})
		}
	}
}
