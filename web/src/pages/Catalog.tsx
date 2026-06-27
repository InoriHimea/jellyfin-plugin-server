import { useEffect, useState } from 'react'
import { api, type CatalogEntry } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { toast } from 'sonner'
import { Download, Search, RefreshCw, CheckCircle2, Loader2, Clock, AlertCircle, Package } from 'lucide-react'

// Category → color
const CAT_COLOR: Record<string, string> = {
  'Metadata':       'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300',
  'Subtitles':      'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300',
  'Sync':           'bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-300',
  'Notifications':  'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-300',
  'UI':             'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300',
  'Live TV':        'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300',
  'Authentication': 'bg-pink-100 text-pink-700 dark:bg-pink-900/40 dark:text-pink-300',
  'General':        'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400',
}
const catColor = (cat: string) => CAT_COLOR[cat] ?? 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400'

// Status config
const STATUS: Record<string, { label: string; icon: React.ElementType; className: string }> = {
  done:        { label: '已下载', icon: CheckCircle2, className: 'text-emerald-600 dark:text-emerald-400' },
  downloading: { label: '下载中', icon: Loader2,      className: 'text-blue-500 animate-spin' },
  pending:     { label: '待下载', icon: Clock,         className: 'text-slate-400' },
  failed:      { label: '失败',   icon: AlertCircle,   className: 'text-red-500' },
  '':          { label: '未知',   icon: Clock,         className: 'text-slate-400' },
}

export function Catalog() {
  const [entries, setEntries] = useState<CatalogEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [downloading, setDownloading] = useState<Set<string>>(new Set())
  const [search, setSearch] = useState('')
  const [activeCategory, setActiveCategory] = useState('全部')

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
    const matchSearch = !q || e.name.toLowerCase().includes(q) || e.owner.toLowerCase().includes(q) || e.description.toLowerCase().includes(q)
    return matchCat && matchSearch
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
              onClick={() => setActiveCategory(cat)}
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

      {/* Plugin grid */}
      {loading ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="h-36 rounded-xl bg-muted/50 animate-pulse" />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground gap-3">
          <Package className="h-12 w-12 opacity-30" />
          <p className="text-sm">没有匹配的插件</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
          {filtered.map(e => {
            const cat = e.category || 'General'
            const status = STATUS[e.latest_status] ?? STATUS['']
            const StatusIcon = status.icon
            const isDoing = downloading.has(e.guid)
            return (
              <Card key={e.guid} className="group overflow-hidden border-border/60 hover:border-primary/30 hover:shadow-md transition-all duration-200 shadow-sm">
                <div className="h-0.5 bg-gradient-to-r from-primary/60 to-violet-400/60 opacity-0 group-hover:opacity-100 transition-opacity" />
                <CardContent className="p-4 flex flex-col gap-3">
                  <div className="flex items-start gap-3">
                    {/* Avatar */}
                    <div className="h-10 w-10 rounded-xl bg-gradient-to-br from-primary/20 to-violet-500/20 flex items-center justify-center text-xs font-bold text-primary shrink-0 border border-primary/10">
                      {initial(e.name)}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <p className="font-semibold text-sm leading-none">{e.name}</p>
                        <span className={`text-[10px] px-1.5 py-0.5 rounded-full font-medium ${catColor(cat)}`}>{cat}</span>
                      </div>
                      <p className="text-xs text-muted-foreground mt-0.5">{e.owner || e.repo_name}</p>
                    </div>
                  </div>

                  {/* Description */}
                  <p className="text-xs text-muted-foreground line-clamp-2 leading-relaxed min-h-[2.5rem]">
                    {e.description || e.overview || '暂无描述'}
                  </p>

                  {/* Footer */}
                  <div className="flex items-center justify-between pt-1 border-t border-border/50">
                    <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                      <StatusIcon className={`h-3.5 w-3.5 ${status.className}`} />
                      <span>{e.latest_version ? `v${e.latest_version}` : '—'}</span>
                      {e.version_count > 1 && (
                        <span className="text-[10px] opacity-60">+{e.version_count - 1}</span>
                      )}
                    </div>
                    <Button
                      size="sm"
                      variant={e.latest_status === 'done' ? 'outline' : 'default'}
                      className="h-7 text-xs gap-1.5"
                      disabled={isDoing || e.latest_status === 'downloading'}
                      onClick={() => triggerDownload(e)}
                    >
                      {isDoing || e.latest_status === 'downloading'
                        ? <><Loader2 className="h-3 w-3 animate-spin" />下载中</>
                        : e.latest_status === 'done'
                          ? <><CheckCircle2 className="h-3 w-3" />重新下载</>
                          : <><Download className="h-3 w-3" />下载</>}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}
