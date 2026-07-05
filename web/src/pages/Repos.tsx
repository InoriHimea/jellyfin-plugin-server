import { useEffect, useState } from 'react'
import { api, type Repo } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table'
import { toast } from 'sonner'
import { Plus, RefreshCw, Trash2, Wifi, WifiOff, Pencil, Sparkles } from 'lucide-react'

const emptyForm = { name: '', url: '', priority: 50, enabled: true }

export function Repos() {
  const [repos, setRepos] = useState<Repo[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState<string | null>(null)
  const [testing, setTesting] = useState<string | null>(null)
  const [dlgOpen, setDlgOpen] = useState(false)
  const [editing, setEditing] = useState<Repo | null>(null)
  const [form, setForm] = useState(emptyForm)
  const [saving, setSaving] = useState(false)

  const load = () => {
    api.repos.list().then(setRepos).finally(() => setLoading(false))
  }

  useEffect(load, [])

  const openAdd = () => { setEditing(null); setForm(emptyForm); setDlgOpen(true) }
  const openEdit = (r: Repo) => {
    setEditing(r)
    setForm({ name: r.name, url: r.url, priority: r.priority, enabled: r.enabled })
    setDlgOpen(true)
  }

  const save = async () => {
    setSaving(true)
    try {
      if (editing) {
        await api.repos.update(editing.id, form)
        toast.success('仓库已更新')
      } else {
        await api.repos.create({ name: form.name, url: form.url, priority: form.priority })
        toast.success('仓库已添加')
      }
      setDlgOpen(false)
      load()
    } catch (e: unknown) {
      toast.error((e as Error).message)
    } finally {
      setSaving(false)
    }
  }

  const del = async (r: Repo) => {
    if (!confirm(`确认删除「${r.name}」？`)) return
    await api.repos.delete(r.id).catch((e: unknown) => toast.error((e as Error).message))
    toast.success('已删除')
    load()
  }

  const refresh = async (r: Repo) => {
    setRefreshing(r.id)
    try {
      await api.repos.refresh(r.id)
      toast.success(`「${r.name}」已刷新`)
      load()
    } catch (e: unknown) {
      toast.error((e as Error).message)
    } finally {
      setRefreshing(null)
    }
  }

  const test = async (r: Repo) => {
    setTesting(r.id)
    try {
      const res = await api.repos.test(r.id)
      if (res.reachable) toast.success(`可达，状态码 ${res.status_code}`)
      else toast.error(`不可达：${res.error}`)
    } finally {
      setTesting(null)
    }
  }

  const refreshAll = async () => {
    setRefreshing('all')
    try {
      const res = await api.repos.refreshAll()
      const failed = Object.entries(res).filter(([, v]) => v !== 'ok')
      if (failed.length) toast.error(`${failed.length} 个仓库刷新失败`)
      else toast.success('全部仓库刷新完成')
      load()
    } finally {
      setRefreshing(null)
    }
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold flex items-center gap-2">
          仓库管理 <Sparkles className="h-5 w-5 text-sakura sparkle-pulse" />
        </h1>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={refreshAll} disabled={refreshing === 'all'}>
            <RefreshCw className={`h-4 w-4 mr-1 ${refreshing === 'all' ? 'animate-spin' : ''}`} />
            刷新全部
          </Button>
          <Button size="sm" onClick={openAdd}>
            <Plus className="h-4 w-4 mr-1" />添加仓库
          </Button>
        </div>
      </div>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-8 text-center text-muted-foreground">加载中…</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>名称</TableHead>
                  <TableHead>URL</TableHead>
                  <TableHead>优先级</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>最后刷新</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {repos.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell className="font-medium">{r.name}</TableCell>
                    <TableCell className="text-xs text-muted-foreground max-w-xs truncate">{r.url}</TableCell>
                    <TableCell>{r.priority}</TableCell>
                    <TableCell>
                      <Badge className={r.enabled ? 'bg-green-500 text-white hover:bg-green-600' : ''} variant={r.enabled ? 'default' : 'secondary'}>
                        {r.enabled ? '启用' : '禁用'}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {r.last_fetched ? new Date(r.last_fetched).toLocaleString('zh-CN') : '从未'}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Button variant="ghost" size="icon" title="测试连通性" onClick={() => test(r)} disabled={testing === r.id}>
                          {testing === r.id ? <WifiOff className="h-4 w-4 animate-pulse" /> : <Wifi className="h-4 w-4" />}
                        </Button>
                        <Button variant="ghost" size="icon" title="刷新" onClick={() => refresh(r)} disabled={refreshing === r.id}>
                          <RefreshCw className={`h-4 w-4 ${refreshing === r.id ? 'animate-spin' : ''}`} />
                        </Button>
                        <Button variant="ghost" size="icon" title="编辑" onClick={() => openEdit(r)}>
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="icon" title="删除" onClick={() => del(r)} className="text-destructive hover:text-destructive">
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={dlgOpen} onOpenChange={setDlgOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? '编辑仓库' : '添加仓库'}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-1">
              <Label>名称</Label>
              <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="My Plugin Repo" />
            </div>
            <div className="space-y-1">
              <Label>Manifest URL</Label>
              <Input value={form.url} onChange={(e) => setForm({ ...form, url: e.target.value })} placeholder="https://example.com/manifest.json" />
            </div>
            <div className="space-y-1">
              <Label>优先级（数字越大越优先）</Label>
              <Input type="number" value={form.priority} onChange={(e) => setForm({ ...form, priority: +e.target.value })} />
            </div>
            <div className="flex items-center gap-2">
              <Switch checked={form.enabled} onCheckedChange={(v) => setForm({ ...form, enabled: v })} />
              <Label>启用</Label>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDlgOpen(false)}>取消</Button>
            <Button onClick={save} disabled={saving || !form.name || !form.url}>
              {saving ? '保存中…' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
