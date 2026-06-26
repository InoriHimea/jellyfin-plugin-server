export interface Repo {
  id: string
  name: string
  url: string
  enabled: boolean
  priority: number
  last_fetched?: string
  etag?: string
  created_at: string
}

export interface Package {
  id: string
  name: string
  owner: string
  version: string
  checksum: string
  status: 'pending' | 'downloading' | 'done' | 'failed'
  local_path?: string
  source_url: string
  downloaded_at?: string
}

export interface LogEntry {
  id: number
  level: string
  message: string
  detail?: string
  created_at: string
}

export interface Status {
  status: string
  version: string
  uptime: string
  db_ok: boolean
  disk_used_mb: number
}

export interface CleanResult {
  lru_removed: string[]
  orphan_removed: string[]
  bytes_freed: number
  dry_run: boolean
}

export interface Config {
  server: { host: string; port: number }
  storage: { data_dir: string; max_disk_mb: number; keep_versions: number; cleanup_schedule: string }
  cache: { manifest_ttl_seconds: number; max_concurrent_downloads: number }
  proxy: { type: string; address: string; username: string; password: string; no_proxy: string }
  auth: { enabled: boolean; username: string; password: string }
  log_json: boolean
}

async function req<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, options)
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || res.statusText)
  }
  return res.json()
}

export const api = {
  status: () => req<Status>('/api/status'),

  repos: {
    list: () => req<Repo[]>('/api/repos'),
    create: (data: { name: string; url: string; priority: number }) =>
      req<Repo>('/api/repos', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) }),
    update: (id: string, data: Partial<Repo>) =>
      req('/api/repos/' + id, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) }),
    delete: (id: string) => req('/api/repos/' + id, { method: 'DELETE' }),
    refresh: (id: string) => req('/api/repos/' + id + '/refresh', { method: 'POST' }),
    refreshAll: () => req<Record<string, string>>('/api/repos/refresh-all', { method: 'POST' }),
    test: (id: string) => req<{ reachable: boolean; status_code?: number; error?: string }>('/api/repos/' + id + '/test', { method: 'POST' }),
  },

  packages: {
    list: (q?: string) => req<Package[]>('/api/packages' + (q ? '?q=' + encodeURIComponent(q) : '')),
    delete: (checksum: string) => req('/api/packages/' + checksum, { method: 'DELETE' }),
    cleanupPreview: () => req<CleanResult>('/api/packages/cleanup?dry_run=true', { method: 'POST' }),
    cleanup: () => req<CleanResult>('/api/packages/cleanup', { method: 'POST' }),
  },

  logs: {
    list: () => req<LogEntry[]>('/api/logs'),
  },

  config: {
    get: () => req<Config>('/api/config'),
    update: (data: Config) => req('/api/config', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) }),
  },
}
