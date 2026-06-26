import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { api, type Status, type Package } from '@/lib/api'
import { HardDrive, Database, Package as PkgIcon, Wifi, Copy, Check } from 'lucide-react'
import { toast } from 'sonner'

export function Dashboard() {
  const [status, setStatus] = useState<Status | null>(null)
  const [packages, setPackages] = useState<Package[]>([])
  const [repoCount, setRepoCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [copied, setCopied] = useState(false)

  const manifestURL = `${window.location.origin}/manifest`

  const copyManifestURL = () => {
    navigator.clipboard.writeText(manifestURL).then(() => {
      setCopied(true)
      toast.success('已复制到剪贴板')
      setTimeout(() => setCopied(false), 2000)
    })
  }

  useEffect(() => {
    Promise.all([
      api.status(),
      api.packages.list(),
      api.repos.list(),
    ]).then(([s, pkgs, repos]) => {
      setStatus(s)
      setPackages(pkgs ?? [])
      setRepoCount(repos?.length ?? 0)
    }).finally(() => setLoading(false))
  }, [])

  const byStatus = packages.reduce<Record<string, number>>((acc, p) => {
    acc[p.status] = (acc[p.status] ?? 0) + 1
    return acc
  }, {})

  const cards = [
    {
      title: '磁盘使用',
      value: loading ? '—' : `${status?.disk_used_mb ?? 0} MB`,
      icon: HardDrive,
      desc: '已缓存包体积',
    },
    {
      title: '已索引版本',
      value: loading ? '—' : packages.length,
      icon: PkgIcon,
      desc: `已下载 ${byStatus['done'] ?? 0} / 下载中 ${byStatus['downloading'] ?? 0}`,
    },
    {
      title: '上游仓库',
      value: loading ? '—' : repoCount,
      icon: Database,
      desc: '已配置仓库数量',
    },
    {
      title: '服务状态',
      value: loading ? '—' : status?.status === 'ok' ? '正常' : '异常',
      icon: Wifi,
      desc: `运行时长 ${status?.uptime ?? '—'}`,
    },
  ]

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">仪表盘</h1>
        <p className="text-muted-foreground text-sm mt-1">服务器版本 {status?.version ?? '—'}</p>
      </div>

      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {cards.map((c) => (
          <Card key={c.title}>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">{c.title}</CardTitle>
              <c.icon className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{c.value}</div>
              <p className="text-xs text-muted-foreground mt-1">{c.desc}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card className="border-primary/40 bg-primary/5">
        <CardHeader className="pb-2">
          <CardTitle className="text-base">Jellyfin 插件仓库地址</CardTitle>
          <p className="text-xs text-muted-foreground">
            复制以下地址，添加到 Jellyfin → 控制台 → 插件 → 存储库
          </p>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-2 bg-muted rounded-md px-3 py-2">
            <code className="text-sm flex-1 break-all select-all">{manifestURL}</code>
            <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0" onClick={copyManifestURL}>
              {copied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">下载状态分布</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex gap-6">
            {[
              { label: '待下载', key: 'pending', color: 'bg-gray-400' },
              { label: '下载中', key: 'downloading', color: 'bg-blue-500' },
              { label: '已完成', key: 'done', color: 'bg-green-500' },
              { label: '失败', key: 'failed', color: 'bg-red-500' },
            ].map(({ label, key, color }) => (
              <div key={key} className="flex items-center gap-2">
                <div className={`w-3 h-3 rounded-full ${color}`} />
                <span className="text-sm">{label}</span>
                <span className="text-sm font-medium">{byStatus[key] ?? 0}</span>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
