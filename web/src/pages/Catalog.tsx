import { useEffect, useState, useRef } from 'react'
import { api, type CatalogEntry } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { toast } from 'sonner'
import {
  Download, Search, RefreshCw, CheckCircle2, Loader2, Clock,
  AlertCircle, Package, ChevronDown, ChevronRight, Layers,
} from 'lucide-react'

const PAGE = 20

const CAT_COLOR: Record<string, string> = {
  MoviesAndShows: 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300',
  Subtitles:      'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300',
  LiveTV:         'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300',
  Administration: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-300',
  Music:          'bg-pink-100 text-pink-700 dark:bg-pink-900/40 dark:text-pink-300',
  Books:          'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300',
  Anime:          'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300',
  General:        'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400',
  Metadata:       'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300',
}
const catColor = (cat: string) =>
  CAT_COLOR[cat] ?? 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400'

const STATUS_CFG = {
  done:        { label: '已下载', icon: CheckCircle2, cls: 'text-emerald-500 dark:text-emerald-400' },
  downloading: { label: '下载中', icon: Loader2,      cls: 'text-blue-500 animate-spin' },
  pending:     { label: '待下载', icon: Clock,         cls: 'text-slate-400' },
  failed:      { label: '失败',   icon: AlertCircle,   cls: 'text-red-500' },
  '':          { label: '—',      icon: Clock,         cls: 'text-slate-300' },
} as const

type StatusKey = keyof typeof STATUS_CFG

export function Catalog() {
  const [entries, setEntries]       = useState<CatalogEntry[]>([])
  const [loading, setLoading]       = useState(true)
  const [downloading, setDownloading] = useState<Set<string>>(new Set())
  const [search, setSearch]         = useState('')
  const [activeCategory, setActiveCat] = useState('全部')
  const [expanded, setExpanded]     = useState<Set<string>>(new Set())
  const [page, setPage]             = useState(1)
  const sentinelRef                 = useRef<HTMLDivElement>(null)

  const load = () => {
    setLoading(true)
    api.catalog.list().then(setEntries).finally(() => setLoading(false))
  }
  useEffect(load, [])

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

  const toggle = (guid: string) =>
    setExpanded(prev => {
      const next = new Set(prev)
      next.has(guid) ? next.delete(guid) : next.add(guid)
      return next
    })

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

  const initial = (name: string) => name.slice(0, 2).toUpperCase()

  return (
    <div className="p-6 space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">插件目录</h1>
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

      {/* List */}
      {loading ? (
        <div className="space-y-2">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="h-14 rounded-lg bg-muted/50 animate-pulse" />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground gap-3">
          <Package className="h-12 w-12 opacity-30" />
          <p className="text-sm">没有匹配的插件</p>
        </div>
      ) : (
        <div className="rounded-xl border border-border/60 overflow-hidden shadow-sm divide-y divide-border/50">
          {visible.map(e => {
            const cat     = e.category || 'General'
            const isOpen  = expanded.has(e.guid)
            const isDoing = downloading.has(e.guid)
            const status  = STATUS_CFG[(e.latest_status || '') as StatusKey] ?? STATUS_CFG['']
            const StatusIcon = status.icon

            return (
              <div key={e.guid} className="bg-card">
                {/* Row */}
                <div
                  className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-muted/30 transition-colors select-none"
                  onClick={() => toggle(e.guid)}
                >
                  {/* Chevron */}
                  <div className="text-muted-foreground shrink-0">
                    {isOpen
                      ? <ChevronDown className="h-4 w-4" />
                      : <ChevronRight className="h-4 w-4" />
                    }
                  </div>

                  {/* Icon */}
                  <div className="h-8 w-8 rounded-lg bg-gradient-to-br from-primary/20 to-violet-500/20 flex items-center justify-center text-[10px] font-bold text-primary shrink-0 border border-primary/10">
                    {initial(e.name)}
                  </div>

                  {/* Name + category */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-sm font-medium leading-none">{e.name}</span>
                      <span className={`text-[10px] px-1.5 py-0.5 rounded-full font-medium ${catColor(cat)}`}>
                        {cat}
                      </span>
                    </div>
                    <p className="text-xs text-muted-foreground mt-0.5 truncate">{e.owner || e.repo_name}</p>
                  </div>

                  {/* Version + status */}
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground shrink-0">
                    <StatusIcon className={`h-3.5 w-3.5 ${status.cls}`} />
                    <span className="font-mono">{e.latest_version ? `v${e.latest_version}` : '—'}</span>
                    {e.version_count > 1 && (
                      <span className="flex items-center gap-0.5 opacity-50">
                        <Layers className="h-3 w-3" />
                        {e.version_count}
                      </span>
                    )}
                  </div>

                  {/* Download button */}
                  <Button
                    size="sm"
                    variant={e.latest_status === 'done' ? 'outline' : 'default'}
                    className="h-7 text-xs gap-1.5 shrink-0"
                    disabled={isDoing || e.latest_status === 'downloading'}
                    onClick={ev => { ev.stopPropagation(); triggerDownload(e) }}
                  >
                    {isDoing || e.latest_status === 'downloading'
                      ? <><Loader2 className="h-3 w-3 animate-spin" />下载中</>
                      : e.latest_status === 'done'
                        ? <><CheckCircle2 className="h-3 w-3" />重新下载</>
                        : <><Download className="h-3 w-3" />下载</>
                    }
                  </Button>
                </div>

                {/* Expanded details */}
                {isOpen && (
                  <div className="px-4 pb-4 pt-1 ml-7 space-y-3 border-t border-border/30 bg-muted/10">
                    {/* Description */}
                    {(e.description || e.overview) && (
                      <p className="text-sm text-muted-foreground leading-relaxed">
                        {e.description || e.overview}
                      </p>
                    )}

                    {/* Meta */}
                    <div className="flex flex-wrap gap-x-6 gap-y-1 text-xs text-muted-foreground">
                      {e.owner && (
                        <span><span className="font-medium text-foreground/60">作者</span>：{e.owner}</span>
                      )}
                      {e.repo_name && (
                        <span><span className="font-medium text-foreground/60">来源</span>：{e.repo_name}</span>
                      )}
                      {e.version_count > 1 && (
                        <span>
                          <span className="font-medium text-foreground/60">版本数</span>：{e.version_count} 个可用，最新 v{e.latest_version}
                        </span>
                      )}
                    </div>
                  </div>
                )}
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
    </div>
  )
}
