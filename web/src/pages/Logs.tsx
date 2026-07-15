import { useCallback, useEffect, useRef, useState } from 'react'
import { api, type LogEntry } from '@/lib/api'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useDebounce } from '@/hooks/use-debounce'
import { ChevronLeft, ChevronRight, RefreshCw, ScrollText, Search, Sparkles } from 'lucide-react'

const PAGE_SIZE = 50

const LEVEL_VARIANT: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  INFO: 'secondary',
  WARN: 'outline',
  ERROR: 'destructive',
  DEBUG: 'outline',
}

const TYPES: { value: string; label: string }[] = [
  { value: '', label: '全部' },
  { value: 'auth', label: '登录' },
  { value: 'access', label: '访问' },
  { value: 'system', label: '系统' },
]

const LEVELS: { value: string; label: string }[] = [
  { value: '', label: '全部级别' },
  { value: 'INFO', label: 'INFO' },
  { value: 'WARN', label: 'WARN' },
  { value: 'ERROR', label: 'ERROR' },
]

const TYPE_LABEL: Record<string, string> = { auth: '登录', access: '访问', system: '系统' }
const TYPE_CLS: Record<string, string> = {
  auth:   'bg-sakura/15 text-pink-700 dark:text-pink-300',
  access: 'bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-300',
  system: 'bg-muted text-muted-foreground',
}

export function Logs() {
  const [entries, setEntries] = useState<LogEntry[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [q, setQ] = useState('')
  const [type, setType] = useState('')
  const [level, setLevel] = useState('')
  const [page, setPage] = useState(0) // zero-based
  const [expanded, setExpanded] = useState<Set<number>>(new Set())

  const debouncedQ = useDebounce(q)

  const load = useCallback(() => {
    setLoading(true)
    api.logs
      .list({ q: debouncedQ, type, level, offset: page * PAGE_SIZE, limit: PAGE_SIZE })
      .then((r) => { setEntries(r.entries ?? []); setTotal(r.total ?? 0) })
      .finally(() => setLoading(false))
  }, [debouncedQ, type, level, page])

  // Single effect drives fetching. When the filter changes while on page >0,
  // reset to page 0 and let the page-change re-run fetch — without this
  // guard the old page number fires one wasted request against the new
  // filter before the reset lands.
  const filterKey = `${debouncedQ}|${type}|${level}`
  const lastFilterRef = useRef(filterKey)
  useEffect(() => {
    if (lastFilterRef.current !== filterKey) {
      lastFilterRef.current = filterKey
      if (page !== 0) { setPage(0); return }
    }
    load()
  }, [filterKey, page, load])

  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const toggleExpand = (id: number) =>
    setExpanded(prev => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold flex items-center gap-2">
          审计日志 <Sparkles className="h-5 w-5 text-sakura sparkle-pulse" />
        </h1>
        <Button variant="outline" size="sm" onClick={load}>
          <RefreshCw className={`h-4 w-4 mr-1 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </Button>
      </div>

      <p className="text-sm text-muted-foreground -mt-2">
        登录事件与公网插件请求（/manifest、/plugins/*）自动记录，保留 30 天。
      </p>

      <div className="flex flex-col sm:flex-row gap-3 sm:items-center">
        <div className="relative flex-1 max-w-xs">
          <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            className="pl-8"
            placeholder="搜索消息 / IP / 路径…"
            value={q}
            onChange={e => setQ(e.target.value)}
          />
        </div>
        <div className="flex gap-1.5 flex-wrap">
          {TYPES.map(t => (
            <button
              key={t.value}
              onClick={() => setType(t.value)}
              className={`px-3 py-1 rounded-full text-xs font-medium transition-all border ${
                type === t.value
                  ? 'bg-primary text-primary-foreground border-primary shadow-sm'
                  : 'border-border text-muted-foreground hover:border-primary/50 hover:text-foreground'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>
        <div className="flex gap-1.5 flex-wrap sm:border-l sm:border-border/60 sm:pl-3">
          {LEVELS.map(l => (
            <button
              key={l.value}
              onClick={() => setLevel(l.value)}
              className={`px-3 py-1 rounded-full text-xs font-medium transition-all border ${
                level === l.value
                  ? 'bg-primary text-primary-foreground border-primary shadow-sm'
                  : 'border-border text-muted-foreground hover:border-primary/50 hover:text-foreground'
              }`}
            >
              {l.label}
            </button>
          ))}
        </div>
      </div>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-8 text-center text-muted-foreground">加载中…</div>
          ) : entries.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-muted-foreground gap-3">
              <ScrollText className="h-12 w-12 opacity-30" />
              <p className="text-sm">{debouncedQ || type || level ? '没有匹配的日志' : '暂无日志'}</p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-20">类型</TableHead>
                  <TableHead className="w-24">级别</TableHead>
                  <TableHead>消息</TableHead>
                  <TableHead>详情</TableHead>
                  <TableHead className="w-40">时间</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {entries.map((l) => {
                  const isOpen = expanded.has(l.id)
                  return (
                    <TableRow key={l.id}>
                      <TableCell>
                        <span className={`text-[10px] px-1.5 py-0.5 rounded-full font-medium ${TYPE_CLS[l.type] ?? TYPE_CLS.system}`}>
                          {TYPE_LABEL[l.type] ?? l.type}
                        </span>
                      </TableCell>
                      <TableCell>
                        <Badge variant={LEVEL_VARIANT[l.level] ?? 'outline'}>{l.level}</Badge>
                      </TableCell>
                      <TableCell className="font-medium text-sm">{l.message}</TableCell>
                      <TableCell
                        className={`text-xs text-muted-foreground max-w-xs ${
                          isOpen ? 'whitespace-pre-wrap break-all' : 'truncate'
                        } ${l.detail ? 'cursor-pointer hover:text-foreground transition-colors' : ''}`}
                        title={isOpen ? '点击收起' : '点击展开完整内容'}
                        onClick={() => l.detail && toggleExpand(l.id)}
                      >
                        {l.detail || '—'}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground whitespace-nowrap tabular-nums">
                        {new Date(l.created_at).toLocaleString('zh-CN')}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Pager */}
      {total > 0 && (
        <div className="flex items-center justify-between text-sm text-muted-foreground">
          <span className="tabular-nums">
            共 {total} 条 · 第 {page + 1} / {pageCount} 页
          </span>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={loading || page === 0}
              onClick={() => setPage(p => Math.max(0, p - 1))}
            >
              <ChevronLeft className="h-4 w-4 mr-1" />
              上一页
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={loading || page + 1 >= pageCount}
              onClick={() => setPage(p => p + 1)}
            >
              下一页
              <ChevronRight className="h-4 w-4 ml-1" />
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
