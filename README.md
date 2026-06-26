# Jellyfin Plugin Server

A self-hosted proxy/mirror for Jellyfin plugin repositories. Caches plugin manifests and packages locally, replaces upstream URLs with local ones, and serves a management Web UI.

## Features

- **Manifest proxy** — serves local manifests with replaced package URLs; falls back to upstream for undownloaded packages
- **Background downloader** — concurrent package download with MD5 verification, singleflight deduplication
- **Storage management** — LRU version eviction, disk limit enforcement, orphan file cleanup
- **Upstream health checks** — periodic ping of all configured repositories
- **Management UI** — React + shadcn/ui dashboard: repo CRUD, package search, logs, settings
- **Observability** — structured request logging, cache hit rate tracking, enhanced `/health` endpoint

## Quick Start

### Docker Compose (recommended)

```bash
curl -o docker-compose.yml https://raw.githubusercontent.com/inorihimea/jellyfin-plugin-server/main/docker-compose.yml
docker compose up -d
```

The UI is available at `http://localhost:8080`.

### Binary

Download the latest binary from [Releases](https://github.com/inorihimea/jellyfin-plugin-server/releases) and run:

```bash
./jellyfin-plugin-server
```

### Build from source

Requirements: Go 1.22+, Node.js 20+

```bash
git clone https://github.com/inorihimea/jellyfin-plugin-server
cd jellyfin-plugin-server
make build          # builds UI then Go binary → bin/jellyfin-plugin-server
./bin/jellyfin-plugin-server
```

## Configuration

Settings are loaded in priority order: **environment variables > config file > defaults**.

### Environment variables

| Variable                    | Default      | Description                     |
|-----------------------------|--------------|----------------------------------|
| `JPSERVER_CONFIG`           | `config.json`| Path to JSON config file         |
| `JPSERVER_HOST`             | `0.0.0.0`    | Listen address                   |
| `JPSERVER_PORT`             | `8080`       | Listen port                      |
| `JPSERVER_DATA_DIR`         | `./data`     | Data directory (DB + packages)   |
| `JPSERVER_LOG_JSON`         | `false`      | Emit JSON logs                   |
| `JPSERVER_MAX_DISK_MB`      | `10240`      | Disk usage limit (MB)            |
| `JPSERVER_MAX_CONCURRENT_DL`| `4`          | Max parallel downloads           |

### config.json

```json
{
  "server": { "host": "0.0.0.0", "port": 8080 },
  "storage": {
    "data_dir": "./data",
    "max_disk_mb": 10240,
    "keep_versions": 3
  },
  "cache": {
    "manifest_ttl_seconds": 86400,
    "max_concurrent_downloads": 4
  },
  "proxy": {
    "type": "",
    "address": "",
    "username": "",
    "password": ""
  },
  "auth": { "enabled": false, "username": "", "password": "" }
}
```

## Jellyfin Integration

1. Start the server and confirm it is healthy:
   ```
   curl http://localhost:8080/health
   ```

2. In Jellyfin → Dashboard → Plugins → Repositories, add a new repository:
   ```
   http://<your-server>:8080/plugins/manifest/<repo-id>
   ```
   The `<repo-id>` for the built-in repos is visible in the management UI under **仓库管理** or via:
   ```
   curl http://localhost:8080/api/repos
   ```

3. Jellyfin will fetch the manifest from this server. Packages already cached locally are served directly; others fall back to the upstream URL automatically.

## API Reference

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check (healthy / degraded / unhealthy) |
| GET | `/api/status` | Version, uptime, disk usage |
| GET | `/api/repos` | List repositories |
| POST | `/api/repos` | Create repository |
| PUT | `/api/repos/:id` | Update repository |
| DELETE | `/api/repos/:id` | Delete repository |
| POST | `/api/repos/:id/refresh` | Manually refresh manifest |
| POST | `/api/repos/refresh-all` | Refresh all manifests |
| POST | `/api/repos/:id/test` | Test upstream connectivity |
| GET | `/api/packages` | List cached packages (supports `?q=`) |
| DELETE | `/api/packages/:checksum` | Delete cached package |
| POST | `/api/packages/cleanup` | Run LRU + orphan cleanup (supports `?dry_run=true`) |
| GET | `/api/config` | Get current configuration |
| PUT | `/api/config` | Update configuration |
| GET | `/api/logs` | Recent operation logs |
| GET | `/plugins/manifest/:repo-id` | Serve local manifest |
| GET | `/plugins/packages/:checksum/:filename` | Serve cached package file |

## License

MIT
