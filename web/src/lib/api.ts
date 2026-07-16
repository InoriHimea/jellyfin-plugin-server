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

// failed_permanent: checksum mismatch or a 4xx response — the upstream
// manifest itself is wrong (deleted release, renamed asset, bad declared
// hash), so retrying just re-fetches the same broken thing. Excluded from
// auto-retry; only resets to pending if the upstream checksum changes.
export type DownloadStatus = 'pending' | 'downloading' | 'done' | 'failed' | 'failed_permanent'

export interface Package {
  id: string
  name: string
  owner: string
  version: string
  checksum: string
  status: DownloadStatus
  local_path?: string
  source_url: string
  downloaded_at?: string
  fail_reason?: string
}

export interface LogEntry {
  id: number
  type: string
  level: string
  message: string
  detail?: string
  created_at: string
}

export interface LogsResponse {
  total: number
  entries: LogEntry[]
}

export interface Status {
  status: string
  version: string
  uptime: string
  db_ok: boolean
  disk_used_mb: number
}

export interface CatalogEntry {
  guid: string
  name: string
  description: string
  overview: string
  owner: string
  category: string
  repo_name: string
  image_url?: string
  version_id: string
  latest_version: string
  latest_status: DownloadStatus | ''
  version_count: number
}

export interface CatalogVersionEntry {
  id: string
  version: string
  target_abi: string
  changelog?: string
  checksum: string
  status: DownloadStatus | ''
  timestamp: string
  repo_name: string
}

export interface CleanResult {
  lru_removed: string[]
  orphan_removed: string[]
  bytes_freed: number
  dry_run: boolean
}

export interface ActiveDownload {
  version_id: string
  checksum: string
  filename: string
  done_bytes: number
  total_bytes: number
  percent: number
  speed_bps: number
  elapsed_sec: number
  name: string
  version: string
}

export interface DownloadsStatus {
  summary: { pending: number; downloading: number; done: number; failed: number; failed_permanent: number; total: number }
  active: ActiveDownload[]
}

export interface Config {
  server: { host: string; port: number; public_url: string }
  storage: { data_dir: string; max_disk_mb: number; keep_versions: number; cleanup_schedule: string }
  cache: { manifest_ttl_seconds: number; max_concurrent_downloads: number }
  proxy: { type: string; address: string; username: string; password: string; no_proxy: string }
  auth: { enabled: boolean; username: string; password: string }
  log_json: boolean
}

const TOKEN_KEY = 'jpserver_token'

export const token = {
  get: () => localStorage.getItem(TOKEN_KEY) ?? '',
  set: (t: string) => localStorage.setItem(TOKEN_KEY, t),
  clear: () => localStorage.removeItem(TOKEN_KEY),
}

async function req<T>(path: string, options?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {
    ...(options?.headers as Record<string, string>),
  }
  const t = token.get()
  if (t) headers['Authorization'] = `Bearer ${t}`

  const res = await fetch(path, { ...options, headers })

  if (res.status === 401) {
    token.clear()
    window.location.href = '/login'
    throw new Error('Session expired')
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || res.statusText)
  }
  return res.json()
}

export const api = {
  status: () => req<Status>('/api/status'),

  auth: {
    status: () => fetch('/api/auth/status').then(r => r.json()) as Promise<{ enabled: boolean }>,
    login: (username: string, password: string) =>
      fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      }).then(async r => {
        const data = await r.json()
        if (!r.ok) throw new Error(data.error || r.statusText)
        return data as { token: string }
      }),
    logout: () => req('/api/auth/logout', { method: 'POST' }),
  },

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

  downloads: {
    status: () => req<DownloadsStatus>('/api/downloads/status'),
    retryFailed: () => req<{ retrying: number }>('/api/downloads/retry-failed', { method: 'POST' }),
  },

  packages: {
    list: (q?: string) => req<Package[]>('/api/packages' + (q ? '?q=' + encodeURIComponent(q) : '')),
    delete: (checksum: string) => req('/api/packages/' + checksum, { method: 'DELETE' }),
    cleanupPreview: () => req<CleanResult>('/api/packages/cleanup?dry_run=true', { method: 'POST' }),
    cleanup: () => req<CleanResult>('/api/packages/cleanup', { method: 'POST' }),
  },

  logs: {
    list: (opts: { q?: string; type?: string; level?: string; offset?: number; limit?: number } = {}) => {
      const params = new URLSearchParams()
      if (opts.q) params.set('q', opts.q)
      if (opts.type) params.set('type', opts.type)
      if (opts.level) params.set('level', opts.level)
      if (opts.offset) params.set('offset', String(opts.offset))
      if (opts.limit) params.set('limit', String(opts.limit))
      const qs = params.toString()
      return req<LogsResponse>('/api/logs' + (qs ? '?' + qs : ''))
    },
  },

  config: {
    get: () => req<Config>('/api/config'),
    update: (data: Config) => req('/api/config', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) }),
  },

  catalog: {
    list: () => req<CatalogEntry[]>('/api/catalog'),
    versions: (guid: string) => req<CatalogVersionEntry[]>(`/api/catalog/${guid}/versions`),
    download: (guid: string) => req<{ status: string }>(`/api/catalog/${guid}/download`, { method: 'POST' }),
  },
}
