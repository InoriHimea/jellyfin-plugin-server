import { useEffect, useState, useCallback } from 'react'
import { api, type Package, type CleanResult } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Card, CardContent } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from '@/components/ui/dialog'
import { toast } from 'sonner'
import { Search, Trash2, RefreshCw, Sparkles } from 'lucide-react'

const STATUS_COLOR: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  done: 'default',
  downloading: 'secondary',
  pending: 'outline',
  failed: 'destructive',
}

const STATUS_LABEL: Record<string, string> = {
  done: '已完成', downloading: '下载中', pending: '待下载', failed: '失败',
}

function fmtBytes(b: number) {
  if (b < 1024 * 1024) return (b / 1024).toFixed(1) + ' KB'
  return (b / 1024 / 1024).toFixed(1) + ' MB'
}

export function Packages() {
  const [packages, setPackages] = useState<Package[]>([])
  const [q, setQ] = useState('')
  const [loading, setLoading] = useState(true)

  const [cleanDlgOpen, setCleanDlgOpen] = useState(false)
  const [preview, setPreview] = useState<CleanResult | null>(null)
  const [previewing, setPreviewing] = useState(false)
  const [cleaning, setCleaning] = useState(false)

  const load = useCallback(() => {
    setLoading(true)
    api.packages.list(q).then((p) => setPackages(p ?? [])).finally(() => setLoading(false))
  }, [q])

  useEffect(() => { load() }, [load])

  const del = async (pkg: Package) => {
    if (!confirm(`确认删除「${pkg.name} ${pkg.version}」的本地缓存？`)) return
    try {
      await api.packages.delete(pkg.checksum)
      toast.success('已删除本地缓存')
      load()
    } catch (e: unknown) {
      toast.error((e as Error).message)
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

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">插件包</h1>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={openCleanDialog}>
            <Sparkles className="h-4 w-4 mr-1" />
            立即清理
          </Button>
          <Button variant="outline" size="sm" onClick={load}>
            <RefreshCw className={`h-4 w-4 mr-1 ${loading ? 'animate-spin' : ''}`} />
            刷新
          </Button>
        </div>
      </div>

      <div className="relative max-w-sm">
        <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
        <Input
          className="pl-8"
          placeholder="搜索插件名…"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </div>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-8 text-center text-muted-foreground">加载中…</div>
          ) : packages.length === 0 ? (
            <div className="p-8 text-center text-muted-foreground">暂无数据</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>插件名</TableHead>
                  <TableHead>作者</TableHead>
                  <TableHead>版本</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>下载时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {packages.map((p) => (
                  <TableRow key={p.id}>
                    <TableCell className="font-medium">{p.name}</TableCell>
                    <TableCell className="text-muted-foreground text-sm">{p.owner || '—'}</TableCell>
                    <TableCell className="font-mono text-sm">{p.version}</TableCell>
                    <TableCell>
                      <Badge variant={STATUS_COLOR[p.status] ?? 'outline'}>
                        {STATUS_LABEL[p.status] ?? p.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {p.downloaded_at ? new Date(p.downloaded_at).toLocaleString('zh-CN') : '—'}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="icon"
                        disabled={p.status !== 'done'}
                        onClick={() => del(p)}
                        className="text-destructive hover:text-destructive disabled:opacity-30"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
      <p className="text-xs text-muted-foreground">共 {packages.length} 条记录</p>

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
