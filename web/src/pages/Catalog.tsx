import { useEffect, useState, useRef } from 'react'
import { api, type CatalogEntry, type CatalogVersionEntry } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from '@/components/ui/dialog'
import { toast } from 'sonner'
import {
  Download, Search, RefreshCw, CheckCircle2, Loader2, Clock,
  AlertCircle, AlertTriangle, Package, Layers, Sparkles,
} from 'lucide-react'

const PAGE = 24

const CAT_COLOR: Record<string, string> = {
  MoviesAndShows: 'bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-300',
  Subtitles:      'bg-mint/20 text-emerald-700 dark:text-emerald-300',
  LiveTV:         'bg-destructive/15 text-destructive dark:text-red-300',
  Administration: 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300',
  Music:          'bg-sakura/15 text-pink-700 dark:text-pink-300',
  Books:          'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300',
  Anime:          'bg-lavender/20 text-violet-700 dark:text-violet-300',
  General:        'bg-muted text-muted-foreground',
  Metadata:       'bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-300',
}
const catColor = (cat: string) =>
  CAT_COLOR[cat] ?? 'bg-muted text-muted-foreground'

const STATUS_CFG = {
  done:             { label: '已下载',   icon: CheckCircle2, cls: 'text-mint' },
  downloading:      { label: '下载中',   icon: Loader2,      cls: 'text-lavender animate-spin' },
  pending:          { label: '待下载',   icon: Clock,         cls: 'text-muted-foreground' },
  failed:           { label: '失败',     icon: AlertCircle,   cls: 'text-destructive' },
  failed_permanent: { label: '永久失败', icon: AlertCircle,   cls: 'text-muted-foreground/60' },
  '':               { label: '—',        icon: Clock,         cls: 'text-muted-foreground/50' },
} as const

type StatusKey = keyof typeof STATUS_CFG

function PluginIcon({ guid, imageUrl, name, size = 'md' }: { guid: string; imageUrl?: string; name: string; size?: 'md' | 'lg' }) {
  const [failed, setFailed] = useState(false)
  const dims = size === 'lg' ? 'h-16 w-16 text-lg' : 'h-11 w-11 text-xs'

  if (imageUrl && !failed) {
    return (
      <img
        src={`/plugins/images/${guid}`}
        alt={name}
        loading="lazy"
        onError={() => setFailed(true)}
        className={`${dims} rounded-2xl object-cover border border-border/40 shrink-0 bg-muted`}
      />
    )
  }
  return (
    <div className={`${dims} rounded-2xl bg-gradient-to-br from-sakura/20 to-lavender/20 flex items-center justify-center font-bold text-sakura shrink-0 border border-sakura/10`}>
      {name.slice(0, 2).toUpperCase()}
    </div>
  )
}

export function Catalog() {
  const [entries, setEntries]       = useState<CatalogEntry[]>([])
  const [loading, setLoading]       = useState(true)
  const [downloading, setDownloading] = useState<Set<string>>(new Set())
  const [search, setSearch]         = useState('')
  const [activeCategory, setActiveCat] = useState('全部')
  const [detail, setDetail]         = useState<CatalogEntry | null>(null)
  const [versions, setVersions]     = useState<CatalogVersionEntry[]>([])
  const [versionsLoading, setVersionsLoading] = useState(false)
  const [page, setPage]             = useState(1)
  const sentinelRef                 = useRef<HTMLDivElement>(null)

  const load = () => {
    setLoading(true)
    api.catalog.list().then(setEntries).finally(() => setLoading(false))
  }
  useEffect(load, [])

  // Fetch the real per-version list on demand — the card only carries a
  // "latest version" summary, not the full history.
  useEffect(() => {
    if (!detail) { setVersions([]); return }
    setVersionsLoading(true)
    api.catalog.versions(detail.guid)
      .then(setVersions)
      .finally(() => setVersionsLoading(false))
  }, [detail])

  const categories = ['全部', ...Array.from(new Set(entries.map(e => e.category || 'General'))).sort()]

  const filtered = entries.filter(e => {
    const cat = e.category || 'General'
    const matchCat = activeCategory === '全部' || cat === activeCategory
    const q = search.toLowerCase()
    const matchSearch =
      !q ||
      e.name.toLowerCase().includes(q) ||
      (e.owner || '').toLowerCase().includes(q) ||
      (e.description || '').toLowerCase().includes(q)
    return matchCat && matchSearch
  })

  const visible  = filtered.slice(0, page * PAGE)
  const hasMore  = visible.length < filtered.length

  // Reset page when filter/search changes
  useEffect(() => setPage(1), [search, activeCategory])

  // IntersectionObserver sentinel
  useEffect(() => {
    const el = sentinelRef.current
    if (!el || !hasMore) return
    const obs = new IntersectionObserver(([e]) => {
      if (e.isIntersecting) setPage(p => p + 1)
    }, { rootMargin: '300px' })
    obs.observe(el)
    return () => obs.disconnect()
  }, [hasMore, visible.length])

  const triggerDownload = async (e: CatalogEntry) => {
    setDownloading(prev => new Set(prev).add(e.guid))
    try {
      await api.catalog.download(e.guid)
      toast.success(`「${e.name}」已加入下载队列`)
      setTimeout(load, 1500)
    } catch (err: unknown) {
      toast.error((err as Error).message)
    } finally {
      setDownloading(prev => { const s = new Set(prev); s.delete(e.guid); return s })
    }
  }

  return (
    <div className="p-6 space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            插件目录 <Sparkles className="h-5 w-5 text-sakura sparkle-pulse" />
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            {loading ? '加载中…' : `共 ${entries.length} 个插件，来自所有已启用仓库`}
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={load} disabled={loading}>
          <RefreshCw className={`h-4 w-4 mr-1.5 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </Button>
      </div>

      {/* Search + category filter */}
      <div className="flex flex-col sm:flex-row gap-3">
        <div className="relative flex-1 max-w-xs">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder="搜索插件名称、作者…"
            value={search}
            onChange={e => setSearch(e.target.value)}
          />
        </div>
        <div className="flex gap-1.5 flex-wrap">
          {categories.map(cat => (
            <button
              key={cat}
              onClick={() => setActiveCat(cat)}
              className={`px-3 py-1 rounded-full text-xs font-medium transition-all border ${
                activeCategory === cat
                  ? 'bg-primary text-primary-foreground border-primary shadow-sm'
                  : 'border-border text-muted-foreground hover:border-primary/50 hover:text-foreground'
              }`}
            >
              {cat}
            </button>
          ))}
        </div>
      </div>

      {/* Card grid */}
      {loading ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="h-[168px] rounded-2xl bg-muted/50 animate-pulse" />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground gap-3">
          <Package className="h-12 w-12 opacity-30" />
          <p className="text-sm">没有匹配的插件</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {visible.map(e => {
            const cat     = e.category || 'General'
            const isDoing = downloading.has(e.guid)
            const status  = STATUS_CFG[(e.latest_status || '') as StatusKey] ?? STATUS_CFG['']
            const StatusIcon = status.icon

            return (
              <div
                key={e.guid}
                onClick={() => setDetail(e)}
                className="group flex flex-col rounded-2xl border border-border/60 bg-card p-4 shadow-soft hover:shadow-md hover:border-primary/40 transition-all cursor-pointer"
              >
                <div className="flex items-start gap-3">
                  <PluginIcon guid={e.guid} imageUrl={e.image_url} name={e.name} />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-semibold leading-tight truncate">{e.name}</p>
                    <p className="text-xs text-muted-foreground mt-0.5 truncate">{e.owner || e.repo_name}</p>
                    <span className={`inline-block mt-1.5 text-[10px] px-1.5 py-0.5 rounded-full font-medium ${catColor(cat)}`}>
                      {cat}
                    </span>
                  </div>
                </div>

                {(e.description || e.overview) && (
                  <p className="text-xs text-muted-foreground mt-3 line-clamp-2 leading-relaxed">
                    {e.description || e.overview}
                  </p>
                )}

                <div className="mt-auto pt-3 flex items-center justify-between">
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <StatusIcon className={`h-3.5 w-3.5 ${status.cls}`} />
                    <span className="font-mono">{e.latest_version ? `v${e.latest_version}` : '—'}</span>
                    {e.latest_version_compatible === false && (
                      <span title="最新版本的 targetAbi 高于系统设置里配置的 Jellyfin 版本，安装后大概率显示 Not Supported">
                        <AlertTriangle className="h-3 w-3 text-amber-500" />
                      </span>
                    )}
                    {e.version_count > 1 && (
                      <span className="flex items-center gap-0.5 opacity-50">
                        <Layers className="h-3 w-3" />
                        {e.version_count}
                      </span>
                    )}
                  </div>
                  <Button
                    size="sm"
                    variant={e.latest_status === 'done' ? 'outline' : 'default'}
                    className="h-7 w-7 p-0 shrink-0"
                    disabled={isDoing || e.latest_status === 'downloading'}
                    onClick={ev => { ev.stopPropagation(); triggerDownload(e) }}
                  >
                    {isDoing || e.latest_status === 'downloading'
                      ? <Loader2 className="h-3.5 w-3.5 animate-spin" />
                      : e.latest_status === 'done'
                        ? <CheckCircle2 className="h-3.5 w-3.5" />
                        : <Download className="h-3.5 w-3.5" />
                    }
                  </Button>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Scroll sentinel + footer */}
      {!loading && filtered.length > 0 && (
        <div className="flex flex-col items-center gap-2 py-2">
          <div ref={sentinelRef} />
          {hasMore ? (
            <p className="text-xs text-muted-foreground animate-pulse">
              加载更多… ({visible.length} / {filtered.length})
            </p>
          ) : (
            <p className="text-xs text-muted-foreground">
              已显示全部 {filtered.length} 个插件
            </p>
          )}
        </div>
      )}

      {/* Detail dialog */}
      <Dialog open={!!detail} onOpenChange={(open) => !open && setDetail(null)}>
        {/*
          grid-cols-[minmax(0,1fr)]: the base DialogContent is `display:grid`
          with no grid-template-columns, so its implicit column defaults to
          `auto` sizing — which grows to fit the widest descendant's
          max-content (text-overflow:ellipsis does NOT reduce max-content
          for a white-space:nowrap element, only how it renders once a
          width is already fixed). Without an explicit track, min-w-0 on
          the version rows has nothing definite to shrink into, so
          truncate silently fails and the dialog's real layout width blows
          out past 5000px, pushing the footer buttons off-screen. minmax(0,1fr)
          gives the track a definite (zero-minimum) basis to resolve against.
        */}
        <DialogContent className="max-w-lg max-h-[85vh] overflow-y-auto grid-cols-[minmax(0,1fr)]">
          {detail && (
            <>
              <DialogHeader>
                <div className="flex items-start gap-3">
                  <PluginIcon guid={detail.guid} imageUrl={detail.image_url} name={detail.name} size="lg" />
                  <div className="min-w-0 pt-1">
                    <DialogTitle>{detail.name}</DialogTitle>
                    <DialogDescription className="mt-1">
                      {detail.owner && <>作者：{detail.owner}　</>}
                      来源：{detail.repo_name}
                    </DialogDescription>
                  </div>
                </div>
              </DialogHeader>

              <div className="space-y-3 py-2">
                <span className={`inline-block text-[10px] px-1.5 py-0.5 rounded-full font-medium ${catColor(detail.category || 'General')}`}>
                  {detail.category || 'General'}
                </span>
                {(detail.description || detail.overview) && (
                  <p className="text-sm text-muted-foreground leading-relaxed">
                    {detail.description || detail.overview}
                  </p>
                )}

                <div>
                  <p className="text-xs font-medium text-foreground/60 mb-1.5">
                    可用版本{versions.length > 0 && <span className="tabular-nums"> · {versions.length} 个</span>}
                  </p>
                  <div className="rounded-xl border border-border/60 divide-y divide-border/50 max-h-64 overflow-y-auto">
                    {versionsLoading ? (
                      <div className="py-6 text-center text-xs text-muted-foreground">加载中…</div>
                    ) : versions.length === 0 ? (
                      <div className="py-6 text-center text-xs text-muted-foreground">没有可用版本</div>
                    ) : (
                      versions.map(v => {
                        const vStatus = STATUS_CFG[(v.status || '') as StatusKey] ?? STATUS_CFG['']
                        const VIcon = vStatus.icon
                        return (
                          <div key={v.id} className="flex items-center gap-2 px-3 py-2 text-xs min-w-0">
                            <span className="font-mono font-medium shrink-0">v{v.version}</span>
                            {v.target_abi && (
                              <span
                                title={v.compatible === false ? '高于系统设置里配置的 Jellyfin 版本，安装后大概率 Not Supported' : undefined}
                                className={`flex items-center gap-1 text-[10px] px-1.5 py-0.5 rounded-full font-medium shrink-0 ${
                                  v.compatible === false
                                    ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
                                    : 'bg-muted text-muted-foreground'
                                }`}
                              >
                                {v.compatible === false && <AlertTriangle className="h-2.5 w-2.5" />}
                                ABI {v.target_abi}
                              </span>
                            )}
                            {/* min-w-0 overrides the flex-item default (min-width:auto), which is
                                what actually lets `truncate` clip instead of forcing the row (and
                                the whole dialog) wider than the container. */}
                            <span className="text-muted-foreground truncate min-w-0 flex-1" title={v.changelog}>
                              {v.changelog || v.repo_name}
                            </span>
                            <VIcon className={`h-3.5 w-3.5 shrink-0 ${vStatus.cls}`} />
                          </div>
                        )
                      })
                    )}
                  </div>
                </div>
              </div>

              <DialogFooter>
                <Button variant="outline" onClick={() => setDetail(null)}>关闭</Button>
                <Button
                  disabled={downloading.has(detail.guid) || detail.latest_status === 'downloading'}
                  onClick={() => triggerDownload(detail)}
                >
                  {downloading.has(detail.guid) || detail.latest_status === 'downloading'
                    ? <><Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />下载中</>
                    : detail.latest_status === 'done'
                      ? <><CheckCircle2 className="h-3.5 w-3.5 mr-1.5" />重新下载</>
                      : <><Download className="h-3.5 w-3.5 mr-1.5" />下载</>
                  }
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
