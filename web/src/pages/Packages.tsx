import { useEffect, useState, useCallback, useRef } from 'react'
import { api, type Package, type CleanResult, type DownloadsStatus } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from '@/components/ui/dialog'
import { toast } from 'sonner'
import {
  Search, Trash2, RefreshCw, Sparkles, ChevronDown, ChevronRight,
  Package as PkgIcon, Download, CheckCircle2, XCircle, Clock, RotateCcw,
} from 'lucide-react'

const PAGE = 20

const STATUS_COLOR: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  done: 'default', downloading: 'secondary', pending: 'outline', failed: 'destructive',
}
const STATUS_LABEL: Record<string, string> = {
  done: '已完成', downloading: '下载中', pending: '待下载', failed: '失败',
}

function fmtBytes(b: number) {
  if (b < 0) return '未知'
  if (b < 1024 * 1024) return (b / 1024).toFixed(1) + ' KB'
  return (b / 1024 / 1024).toFixed(1) + ' MB'
}

function fmtSpeed(bps: number) {
  if (bps <= 0) return '—'
  if (bps < 1024 * 1024) return (bps / 1024).toFixed(0) + ' KB/s'
  return (bps / 1024 / 1024).toFixed(1) + ' MB/s'
}

function fmtDate(s?: string) {
  if (!s) return '—'
  return new Date(s).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

interface PluginGroup {
  name: string
  owner: string
  versions: Package[]
  latestStatus: string
  latestVersion: string
}

function groupPackages(packages: Package[]): PluginGroup[] {
  const map = new Map<string, Package[]>()
  for (const p of packages) {
    const key = p.name
    if (!map.has(key)) map.set(key, [])
    map.get(key)!.push(p)
  }
  return Array.from(map.entries()).map(([name, versions]) => {
    const sorted = [...versions].sort((a, b) => b.version.localeCompare(a.version, undefined, { numeric: true }))
    const latest = sorted[0]
    return { name, owner: latest.owner || '', versions: sorted, latestStatus: latest.status, latestVersion: latest.version }
  })
}

export function Packages() {
  const [packages, setPackages]   = useState<Package[]>([])
  const [q, setQ]                 = useState('')
  const [loading, setLoading]     = useState(true)
  const [expanded, setExpanded]   = useState<Set<string>>(new Set())
  const [page, setPage]           = useState(1)
  const sentinelRef               = useRef<HTMLDivElement>(null)

  const [dl, setDl]               = useState<DownloadsStatus | null>(null)
  const prevDoneRef               = useRef(-1)
  const [retrying, setRetrying]   = useState(false)

  const [cleanDlgOpen, setCleanDlgOpen] = useState(false)
  const [preview, setPreview]           = useState<CleanResult | null>(null)
  const [previewing, setPreviewing]     = useState(false)
  const [cleaning, setCleaning]         = useState(false)

  const load = useCallback(() => {
    setLoading(true)
    api.packages.list(q).then(p => setPackages(p ?? [])).finally(() => setLoading(false))
  }, [q])

  useEffect(() => { load() }, [load])

  // Poll download status; refresh the package list whenever a download lands.
  useEffect(() => {
    let stop = false
    const tick = async () => {
      try {
        const s = await api.downloads.status()
        if (stop) return
        setDl(s)
        if (prevDoneRef.current >= 0 && s.summary.done !== prevDoneRef.current) {
          api.packages.list(q).then(p => { if (!stop) setPackages(p ?? []) })
        }
        prevDoneRef.current = s.summary.done
      } catch { /* server unreachable, keep last state */ }
    }
    tick()
    const iv = setInterval(tick, 2500)
    return () => { stop = true; clearInterval(iv) }
  }, [q])

  const groups = groupPackages(packages)
  const visible = groups.slice(0, page * PAGE)
  const hasMore = visible.length < groups.length

  useEffect(() => setPage(1), [q])

  useEffect(() => {
    const el = sentinelRef.current
    if (!el || !hasMore) return
    const obs = new IntersectionObserver(([e]) => {
      if (e.isIntersecting) setPage(p => p + 1)
    }, { rootMargin: '300px' })
    obs.observe(el)
    return () => obs.disconnect()
  }, [hasMore, visible.length])

  const toggle = (name: string) =>
    setExpanded(prev => {
      const next = new Set(prev)
      next.has(name) ? next.delete(name) : next.add(name)
      return next
    })

  const del = async (pkg: Package) => {
    if (!confirm(`确认删除「${pkg.name} v${pkg.version}」的本地缓存？`)) return
    try {
      await api.packages.delete(pkg.checksum)
      toast.success('已删除本地缓存')
      load()
    } catch (e: unknown) {
      toast.error((e as Error).message)
    }
  }

  const retryFailed = async () => {
    setRetrying(true)
    try {
      const r = await api.downloads.retryFailed()
      toast.success(`已重新排队 ${r.retrying} 个失败的下载`)
    } catch (e: unknown) {
      toast.error((e as Error).message)
    } finally {
      setRetrying(false)
    }
  }

  const openCleanDialog = async () => {
    setPreview(null)
    setCleanDlgOpen(true)
    setPreviewing(true)
    try {
      const result = await api.packages.cleanupPreview()
      setPreview(result)
    } catch (e: unknown) {
      toast.error((e as Error).message)
      setCleanDlgOpen(false)
    } finally {
      setPreviewing(false)
    }
  }

  const runCleanup = async () => {
    setCleaning(true)
    try {
      const result = await api.packages.cleanup()
      setCleanDlgOpen(false)
      toast.success(
        `清理完成：删除 ${(result.lru_removed?.length ?? 0) + (result.orphan_removed?.length ?? 0)} 个文件，释放 ${fmtBytes(result.bytes_freed)}`
      )
      load()
    } catch (e: unknown) {
      toast.error((e as Error).message)
    } finally {
      setCleaning(false)
    }
  }

  const initial = (name: string) => name.slice(0, 2).toUpperCase()

  const sum = dl?.summary
  const cachePct = sum && sum.total > 0 ? Math.round((sum.done / sum.total) * 100) : 0
  const activeById = new Map((dl?.active ?? []).map(a => [a.version_id, a]))

  return (
    <div className="p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">插件包</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {loading ? '加载中…' : `${groups.length} 个插件，共 ${packages.length} 个版本`}
          </p>
        </div>
        <div className="flex gap-2">
          {sum && sum.failed > 0 && (
            <Button variant="outline" size="sm" onClick={retryFailed} disabled={retrying}>
              <RotateCcw className={`h-4 w-4 mr-1 ${retrying ? 'animate-spin' : ''}`} />
              重试失败 ({sum.failed})
            </Button>
          )}
          <Button variant="outline" size="sm" onClick={openCleanDialog}>
            <Sparkles className="h-4 w-4 mr-1" />
            立即清理
          </Button>
          <Button variant="outline" size="sm" onClick={load} disabled={loading}>
            <RefreshCw className={`h-4 w-4 mr-1 ${loading ? 'animate-spin' : ''}`} />
            刷新
          </Button>
        </div>
      </div>

      {/* Cache summary strip */}
      {sum && sum.total > 0 && (
        <div className="rounded-xl border border-border/60 bg-card p-4 space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-5 text-sm flex-wrap">
              <span className="flex items-center gap-1.5">
                <CheckCircle2 className="h-4 w-4 text-emerald-500" />
                <span className="font-medium tabular-nums">{sum.done}</span>
                <span className="text-muted-foreground">已缓存</span>
              </span>
              <span className="flex items-center gap-1.5">
                <Download className={`h-4 w-4 text-blue-500 ${sum.downloading > 0 ? 'animate-pulse' : ''}`} />
                <span className="font-medium tabular-nums">{sum.downloading}</span>
                <span className="text-muted-foreground">下载中</span>
              </span>
              <span className="flex items-center gap-1.5">
                <Clock className="h-4 w-4 text-slate-400" />
                <span className="font-medium tabular-nums">{sum.pending}</span>
                <span className="text-muted-foreground">排队中</span>
              </span>
              <span className="flex items-center gap-1.5">
                <XCircle className="h-4 w-4 text-red-500" />
                <span className="font-medium tabular-nums">{sum.failed}</span>
                <span className="text-muted-foreground">失败</span>
              </span>
            </div>
            <span className="text-sm font-medium tabular-nums">{cachePct}%</span>
          </div>
          <div className="h-2 rounded-full bg-muted overflow-hidden">
            <div
              className="h-full rounded-full bg-emerald-500 transition-all duration-700"
              style={{ width: `${cachePct}%` }}
            />
          </div>
        </div>
      )}

      {/* Active downloads panel */}
      {dl && dl.active.length > 0 && (
        <div className="rounded-xl border border-blue-500/30 bg-blue-500/5 p-4 space-y-3">
          <p className="text-sm font-medium flex items-center gap-2">
            <Download className="h-4 w-4 text-blue-500 animate-pulse" />
            正在下载 {dl.active.length} 个文件
          </p>
          <div className="space-y-2.5">
            {dl.active.map(a => (
              <div key={a.version_id} className="space-y-1">
                <div className="flex items-center justify-between text-xs">
                  <span className="font-medium truncate mr-3">
                    {a.name || a.filename} {a.version && <span className="text-muted-foreground font-mono">v{a.version}</span>}
                  </span>
                  <span className="text-muted-foreground tabular-nums shrink-0">
                    {a.total_bytes > 0
                      ? `${fmtBytes(a.done_bytes)} / ${fmtBytes(a.total_bytes)} · ${fmtSpeed(a.speed_bps)} · ${Math.round(a.percent)}%`
                      : `${fmtBytes(a.done_bytes)} · ${fmtSpeed(a.speed_bps)}`
                    }
                  </span>
                </div>
                <div className="h-1.5 rounded-full bg-muted overflow-hidden">
                  {a.total_bytes > 0 ? (
                    <div
                      className="h-full rounded-full bg-blue-500 transition-all duration-500"
                      style={{ width: `${a.percent}%` }}
                    />
                  ) : (
                    <div className="h-full w-1/3 rounded-full bg-blue-500/60 animate-pulse" />
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Search */}
      <div className="relative max-w-sm">
        <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
        <Input
          className="pl-8"
          placeholder="搜索插件名…"
          value={q}
          onChange={e => setQ(e.target.value)}
        />
      </div>

      {/* Group list */}
      {loading ? (
        <div className="space-y-2">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="h-14 rounded-lg bg-muted/50 animate-pulse" />
          ))}
        </div>
      ) : groups.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground gap-3">
          <PkgIcon className="h-12 w-12 opacity-30" />
          <p className="text-sm">暂无插件包</p>
        </div>
      ) : (
        <div className="rounded-xl border border-border/60 overflow-hidden shadow-sm divide-y divide-border/50">
          {visible.map(group => {
            const isOpen = expanded.has(group.name)
            return (
              <div key={group.name} className="bg-card">
                {/* Group header row */}
                <div
                  className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-muted/30 transition-colors select-none"
                  onClick={() => toggle(group.name)}
                >
                  <div className="text-muted-foreground shrink-0">
                    {isOpen
                      ? <ChevronDown className="h-4 w-4" />
                      : <ChevronRight className="h-4 w-4" />
                    }
                  </div>

                  <div className="h-8 w-8 rounded-lg bg-gradient-to-br from-primary/20 to-violet-500/20 flex items-center justify-center text-[10px] font-bold text-primary shrink-0 border border-primary/10">
                    {initial(group.name)}
                  </div>

                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium leading-none">{group.name}</p>
                    {group.owner && (
                      <p className="text-xs text-muted-foreground mt-0.5 truncate">{group.owner}</p>
                    )}
                  </div>

                  {group.versions.length > 1 && (
                    <span className="text-xs text-muted-foreground shrink-0 tabular-nums">
                      {group.versions.length} 个版本
                    </span>
                  )}

                  <div className="flex items-center gap-2 shrink-0">
                    <span className="text-xs font-mono text-muted-foreground">
                      v{group.latestVersion}
                    </span>
                    <Badge variant={STATUS_COLOR[group.latestStatus] ?? 'outline'} className="text-[10px] h-5">
                      {STATUS_LABEL[group.latestStatus] ?? group.latestStatus}
                    </Badge>
                  </div>
                </div>

                {/* Expanded: version list */}
                {isOpen && (
                  <div className="border-t border-border/30 bg-muted/10">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b border-border/20">
                          <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground pl-[3.25rem]">版本</th>
                          <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">状态</th>
                          <th className="px-4 py-2 text-left text-xs font-medium text-muted-foreground">下载时间</th>
                          <th className="px-4 py-2 text-right text-xs font-medium text-muted-foreground">操作</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-border/20">
                        {group.versions.map(p => {
                          const act = activeById.get(p.id)
                          return (
                            <tr key={p.id} className="hover:bg-muted/20 transition-colors">
                              <td className="px-4 py-2 font-mono text-xs pl-[3.25rem]">v{p.version}</td>
                              <td className="px-4 py-2">
                                {act ? (
                                  <div className="flex items-center gap-2 min-w-[140px]">
                                    <div className="h-1.5 flex-1 rounded-full bg-muted overflow-hidden">
                                      <div
                                        className="h-full rounded-full bg-blue-500 transition-all duration-500"
                                        style={{ width: `${act.total_bytes > 0 ? act.percent : 30}%` }}
                                      />
                                    </div>
                                    <span className="text-[10px] text-muted-foreground tabular-nums shrink-0">
                                      {act.total_bytes > 0 ? `${Math.round(act.percent)}%` : '…'}
                                    </span>
                                  </div>
                                ) : (
                                  <div className="flex flex-col gap-0.5">
                                    <Badge variant={STATUS_COLOR[p.status] ?? 'outline'} className="text-[10px] h-5 w-fit">
                                      {STATUS_LABEL[p.status] ?? p.status}
                                    </Badge>
                                    {p.status === 'failed' && p.fail_reason && (
                                      <span
                                        className="text-[10px] text-red-500/80 max-w-[260px] truncate"
                                        title={p.fail_reason}
                                      >
                                        {p.fail_reason}
                                      </span>
                                    )}
                                  </div>
                                )}
                              </td>
                              <td className="px-4 py-2 text-xs text-muted-foreground tabular-nums">
                                {fmtDate(p.downloaded_at)}
                              </td>
                              <td className="px-4 py-2 text-right">
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-7 w-7 text-destructive/60 hover:text-destructive disabled:opacity-25"
                                  disabled={p.status !== 'done'}
                                  onClick={() => del(p)}
                                >
                                  <Trash2 className="h-3.5 w-3.5" />
                                </Button>
                              </td>
                            </tr>
                          )
                        })}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {/* Scroll sentinel */}
      {!loading && groups.length > 0 && (
        <div className="flex flex-col items-center gap-2 py-1">
          <div ref={sentinelRef} />
          {hasMore ? (
            <p className="text-xs text-muted-foreground animate-pulse">
              加载更多… ({visible.length} / {groups.length})
            </p>
          ) : (
            <p className="text-xs text-muted-foreground">已显示全部 {groups.length} 个插件</p>
          )}
        </div>
      )}

      {/* 清理预览 Dialog */}
      <Dialog open={cleanDlgOpen} onOpenChange={setCleanDlgOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>存储清理</DialogTitle>
            <DialogDescription>
              根据「每个插件保留版本数」配置，删除多余旧版本及孤儿文件。
            </DialogDescription>
          </DialogHeader>

          {previewing ? (
            <div className="py-6 text-center text-muted-foreground text-sm">分析中…</div>
          ) : preview ? (
            <div className="space-y-3 py-2">
              <div className="grid grid-cols-2 gap-3">
                <div className="rounded-lg border p-3">
                  <p className="text-xs text-muted-foreground">旧版本</p>
                  <p className="text-2xl font-bold">{preview.lru_removed?.length ?? 0}</p>
                  <p className="text-xs text-muted-foreground">个文件将被删除</p>
                </div>
                <div className="rounded-lg border p-3">
                  <p className="text-xs text-muted-foreground">孤儿文件</p>
                  <p className="text-2xl font-bold">{preview.orphan_removed?.length ?? 0}</p>
                  <p className="text-xs text-muted-foreground">个文件将被删除</p>
                </div>
              </div>
              <div className="rounded-lg border p-3 bg-muted/30">
                <p className="text-sm font-medium">预计释放空间</p>
                <p className="text-xl font-bold text-green-600 dark:text-green-400">{fmtBytes(preview.bytes_freed)}</p>
              </div>
              {(preview.lru_removed?.length ?? 0) + (preview.orphan_removed?.length ?? 0) === 0 && (
                <p className="text-sm text-center text-muted-foreground py-2">没有需要清理的文件</p>
              )}
            </div>
          ) : null}

          <DialogFooter>
            <Button variant="outline" onClick={() => setCleanDlgOpen(false)}>取消</Button>
            <Button
              onClick={runCleanup}
              disabled={cleaning || previewing || ((preview?.lru_removed?.length ?? 0) + (preview?.orphan_removed?.length ?? 0)) === 0}
              variant="destructive"
            >
              {cleaning ? '清理中…' : '确认清理'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
