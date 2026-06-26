package handler

import (
	"encoding/json"
	"time"

	"github.com/inorihimea/jellyfin-plugin-server/internal/config"
	"github.com/inorihimea/jellyfin-plugin-server/internal/db"
	"github.com/inorihimea/jellyfin-plugin-server/internal/health"
	"github.com/inorihimea/jellyfin-plugin-server/internal/storage"
	"github.com/valyala/fasthttp"
)

var startTime = time.Now()

type healthResponse struct {
	Status      string               `json:"status"`
	Uptime      string               `json:"uptime"`
	Version     string               `json:"version"`
	DBOk        bool                 `json:"db_ok"`
	DiskUsedMB  int64                `json:"disk_used_mb"`
	DiskLimitMB int64                `json:"disk_limit_mb"`
	CacheHitRate float64             `json:"cache_hit_rate_24h"`
	Upstreams   []health.RepoHealth  `json:"upstreams"`
}

func Health(ctx *fasthttp.RequestCtx) {
	dbOk := db.DB != nil && db.DB.Ping() == nil

	cfg := config.Get()
	diskBytes, _ := storage.DiskUsage()
	diskMB := diskBytes / 1024 / 1024

	hitRate, _, _ := db.CacheHitRate(24)

	upstreams := health.Results()

	// Determine overall status:
	//   healthy   — DB ok, all upstreams reachable (or no checks yet)
	//   degraded  — DB ok but some upstreams unreachable
	//   unhealthy — DB unreachable
	status := "healthy"
	code := fasthttp.StatusOK
	if !dbOk {
		status = "unhealthy"
		code = fasthttp.StatusServiceUnavailable
	} else if !health.AllHealthy() && len(upstreams) > 0 {
		status = "degraded"
	}

	resp := healthResponse{
		Status:       status,
		Uptime:       time.Since(startTime).Round(time.Second).String(),
		Version:      Version,
		DBOk:         dbOk,
		DiskUsedMB:   diskMB,
		DiskLimitMB:  cfg.Storage.MaxDiskMB,
		CacheHitRate: hitRate,
		Upstreams:    upstreams,
	}
	b, _ := json.Marshal(resp)
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(code)
	ctx.SetBody(b)
}
