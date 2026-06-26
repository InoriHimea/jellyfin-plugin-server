<div align="center">

# Jellyfin Plugin Server

**A self-hosted proxy & mirror for Jellyfin plugin repositories**

[![CI](https://github.com/inorihimea/jellyfin-plugin-server/actions/workflows/ci.yml/badge.svg)](https://github.com/inorihimea/jellyfin-plugin-server/actions/workflows/ci.yml)
[![GitHub release](https://img.shields.io/github/v/release/inorihimea/jellyfin-plugin-server)](https://github.com/inorihimea/jellyfin-plugin-server/releases/latest)
[![Docker Image](https://img.shields.io/badge/ghcr.io-jellyfin--plugin--server-blue?logo=docker)](https://github.com/inorihimea/jellyfin-plugin-server/pkgs/container/jellyfin-plugin-server)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Cache Jellyfin plugin manifests and packages locally, substitute upstream URLs with local ones, and manage everything through a built-in Web UI.

</div>

---

## Overview

Jellyfin's official plugin CDN can be slow or unreachable depending on your network. **Jellyfin Plugin Server** sits between Jellyfin and the upstream repositories:

```
Jellyfin  →  GET /plugins/manifest/{repo-id}
                ↓
         Jellyfin Plugin Server
          ├─ TTL cache hit  → serve local manifest (URLs already rewritten)
          └─ TTL expired    → fetch upstream, cache, rewrite URLs, respond
                               ↓ (background)
                        download .zip packages → verify MD5 → store locally
```

From then on, all package downloads are served from your local disk at full LAN speed.

## Features

- **Manifest proxy** — rewrites `sourceUrl` in manifest.json to local endpoints; falls back to upstream for packages not yet cached
- **Background downloader** — concurrent downloads with MD5 verification and singleflight deduplication (no duplicate requests for the same file)
- **Built-in repositories** — Jellyfin Official, Intro Skipper, Open Subtitles, Ani-Sync seeded by default
- **Storage management** — LRU version eviction, disk limit guard, orphan file cleanup with dry-run preview
- **Upstream health checks** — periodic connectivity pings; `/health` reports `healthy / degraded / unhealthy`
- **Management Web UI** — dashboard, repo CRUD, package search, structured logs, settings
- **Observability** — per-request structured logging (method / path / status / latency), 24 h cache hit rate
- **HTTP/HTTPS/SOCKS5 proxy** — configurable upstream proxy for air-gapped environments

## Quick Start

### Docker Compose (recommended)

```bash
curl -LO https://raw.githubusercontent.com/inorihimea/jellyfin-plugin-server/main/docker-compose.yml
docker compose up -d
```

Open `http://localhost:8080` — the management UI is ready.

### Pre-built binary

Download from [Releases](https://github.com/inorihimea/jellyfin-plugin-server/releases/latest):

```bash
# Linux amd64 example
curl -L https://github.com/inorihimea/jellyfin-plugin-server/releases/latest/download/jellyfin-plugin-server -o jps
chmod +x jps && ./jps
```

### Build from source

Requirements: **Go 1.22+**, **Node.js 20+**

```bash
git clone https://github.com/inorihimea/jellyfin-plugin-server.git
cd jellyfin-plugin-server
make build        # builds UI then embeds it into the Go binary
./bin/jellyfin-plugin-server
```

## Connecting to Jellyfin

1. **Verify the server is running:**
   ```bash
   curl http://localhost:8080/health
   # {"status":"healthy", ...}
   ```

2. **Get the repo IDs** from the management UI (`/repos`) or the API:
   ```bash
   curl http://localhost:8080/api/repos | jq '.[].id'
   ```

3. **Add a repository in Jellyfin** — Dashboard → Plugins → Repositories → ➕:
   ```
   http://<your-server-ip>:8080/plugins/manifest/<repo-id>
   ```

4. Jellyfin fetches the manifest from this server. Locally cached packages are served at LAN speed; uncached packages transparently redirect to the upstream URL.

## Configuration

Priority: **environment variable > config.json > built-in default**

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `JPSERVER_CONFIG` | `config.json` | Path to the JSON config file |
| `JPSERVER_HOST` | `0.0.0.0` | Listen address |
| `JPSERVER_PORT` | `8080` | Listen port |
| `JPSERVER_DATA_DIR` | `./data` | Data directory (SQLite DB + packages) |
| `JPSERVER_LOG_JSON` | `false` | Output structured JSON logs |
| `JPSERVER_MAX_DISK_MB` | `10240` | Disk usage limit in MB |
| `JPSERVER_MAX_CONCURRENT_DL` | `4` | Maximum parallel downloads |

### config.json

```json
{
  "server":  { "host": "0.0.0.0", "port": 8080 },
  "storage": { "data_dir": "./data", "max_disk_mb": 10240, "keep_versions": 3 },
  "cache":   { "manifest_ttl_seconds": 86400, "max_concurrent_downloads": 4 },
  "proxy":   { "type": "", "address": "", "username": "", "password": "" },
  "auth":    { "enabled": false, "username": "", "password": "" }
}
```

`proxy.type` accepts `""` (none), `"http"`, `"https"`, or `"socks5"`.

## API Reference

<details>
<summary>Expand full API table</summary>

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check — `healthy / degraded / unhealthy`, upstream status |
| `GET` | `/api/status` | Version, git commit, uptime, disk usage |
| `GET` | `/api/repos` | List repositories |
| `POST` | `/api/repos` | Create repository |
| `PUT` | `/api/repos/:id` | Update repository |
| `DELETE` | `/api/repos/:id` | Delete repository |
| `POST` | `/api/repos/:id/refresh` | Force-refresh manifest from upstream |
| `POST` | `/api/repos/refresh-all` | Refresh all enabled repositories |
| `POST` | `/api/repos/:id/test` | Test upstream connectivity |
| `GET` | `/api/packages?q=` | List cached packages (optional search) |
| `DELETE` | `/api/packages/:checksum` | Delete a cached package |
| `POST` | `/api/packages/cleanup?dry_run=true` | LRU + orphan cleanup (dry-run preview) |
| `GET` | `/api/config` | Read current configuration |
| `PUT` | `/api/config` | Update configuration |
| `GET` | `/api/logs` | Last 100 operation log entries |
| `GET` | `/plugins/manifest/:repo-id` | Serve local manifest (Jellyfin endpoint) |
| `GET` | `/plugins/packages/:checksum/:file` | Serve cached package file |

</details>

## Tech Stack

| Layer | Technology |
|-------|-----------|
| HTTP server | [fasthttp](https://github.com/valyala/fasthttp) |
| Upstream client | `net/http` (follows 302 redirects) |
| Database | [modernc/sqlite](https://gitlab.com/cznic/sqlite) — pure Go, no CGO |
| Download dedup | [golang.org/x/sync/singleflight](https://pkg.go.dev/golang.org/x/sync/singleflight) |
| Frontend | React 18 + Vite + [shadcn/ui](https://ui.shadcn.com) + Tailwind CSS |
| Distribution | Embedded via `//go:embed` — single binary, no external assets |

## Contributing

Pull requests are welcome. For significant changes please open an issue first to discuss what you would like to change.

```bash
# Run backend tests
go test ./...

# Start frontend dev server (proxies API to :8080)
cd web && npm run dev
```

## License

[MIT](LICENSE)
