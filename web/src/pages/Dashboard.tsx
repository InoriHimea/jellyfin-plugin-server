import { useEffect, useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { api, type Status, type Package } from '@/lib/api'
import { HardDrive, Database, Package as PkgIcon, Activity, Copy, Check, Zap } from 'lucide-react'
import { toast } from 'sonner'

const STAT_CARDS = (status: Status | null, packages: Package[], repoCount: number, loading: boolean) => {
  const byStatus = packages.reduce<Record<string, number>>((acc, p) => {
    acc[p.status] = (acc[p.status] ?? 0) + 1
    return acc
  }, {})

  return [
    {
      title: '磁盘使用',
      value: loading ? '—' : `${status?.disk_used_mb ?? 0} MB`,
      icon: HardDrive,
      desc: '已缓存包体积',
      iconBg: 'bg-blue-100 dark:bg-blue-900/30',
      iconColor: 'text-blue-600 dark:text-blue-400',
      bar: 'from-blue-500 to-cyan-400',
    },
    {
      title: '已索引版本',
      value: loading ? '—' : packages.length,
      icon: PkgIcon,
      desc: `已下载 ${byStatus['done'] ?? 0} / 待下载 ${byStatus['pending'] ?? 0}`,
      iconBg: 'bg-violet-100 dark:bg-violet-900/30',
      iconColor: 'text-violet-600 dark:text-violet-400',
      bar: 'from-violet-500 to-purple-400',
    },
    {
      title: '上游仓库',
      value: loading ? '—' : repoCount,
      icon: Database,
      desc: '已配置仓库数量',
      iconBg: 'bg-emerald-100 dark:bg-emerald-900/30',
      iconColor: 'text-emerald-600 dark:text-emerald-400',
      bar: 'from-emerald-500 to-teal-400',
    },
    {
      title: '服务状态',
      value: loading ? '—' : status?.status === 'ok' ? '正常' : '异常',
      icon: Activity,
      desc: `运行时长 ${status?.uptime ?? '—'}`,
      iconBg: 'bg-orange-100 dark:bg-orange-900/30',
      iconColor: 'text-orange-600 dark:text-orange-400',
      bar: 'from-orange-500 to-amber-400',
    },
  ]
}

const STATUS_BAR = [
  { label: '待下载', key: 'pending', color: 'bg-slate-300' },
  { label: '下载中', key: 'downloading', color: 'bg-blue-500 animate-pulse' },
  { label: '已完成', key: 'done', color: 'bg-emerald-500' },
  { label: '失败', key: 'failed', color: 'bg-red-500' },
]

export function Dashboard() {
  const [status, setStatus] = useState<Status | null>(null)
  const [packages, setPackages] = useState<Package[]>([])
  const [repoCount, setRepoCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [copied, setCopied] = useState(false)

  const manifestURL = `${window.location.origin}/manifest`

  useEffect(() => {
    Promise.all([api.status(), api.packages.list(), api.repos.list()])
      .then(([s, pkgs, repos]) => {
        setStatus(s)
        setPackages(pkgs ?? [])
        setRepoCount(repos?.length ?? 0)
      })
      .finally(() => setLoading(false))
  }, [])

  const copyURL = () => {
    navigator.clipboard.writeText(manifestURL).then(() => {
      setCopied(true)
      toast.success('已复制到剪贴板')
      setTimeout(() => setCopied(false), 2000)
    })
  }

  const byStatus = packages.reduce<Record<string, number>>((acc, p) => {
    acc[p.status] = (acc[p.status] ?? 0) + 1
    return acc
  }, {})

  const cards = STAT_CARDS(status, packages, repoCount, loading)

  return (
    <div className="p-6 space-y-5">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">仪表盘</h1>
        <p className="text-muted-foreground text-sm mt-1">服务器版本 {status?.version ?? '—'}</p>
      </div>

      {/* Manifest URL — 最醒目卡片 */}
      <div className="relative rounded-xl overflow-hidden border border-blue-200 dark:border-blue-800 bg-gradient-to-br from-blue-50 via-white to-violet-50 dark:from-blue-950/30 dark:via-slate-900 dark:to-violet-950/20 shadow-sm shadow-blue-100 dark:shadow-blue-900/20">
        <div className="absolute top-0 left-0 right-0 h-0.5 bg-gradient-to-r from-blue-500 via-violet-500 to-purple-500" />
        <div className="p-5">
          <div className="flex items-center gap-2 mb-3">
            <div className="h-7 w-7 rounded-lg bg-gradient-to-br from-blue-500 to-violet-600 flex items-center justify-center shadow shadow-blue-500/30">
              <Zap className="h-4 w-4 text-white" />
            </div>
            <div>
              <p className="font-semibold text-sm">Jellyfin 插件仓库地址</p>
              <p className="text-xs text-muted-foreground">添加到 Jellyfin → 控制台 → 插件 → 存储库</p>
            </div>
          </div>
          <div className="flex items-center gap-2 bg-white/80 dark:bg-slate-900/60 backdrop-blur border border-blue-100 dark:border-blue-900 rounded-lg px-4 py-2.5">
            <code className="text-sm flex-1 break-all select-all text-blue-700 dark:text-blue-300 font-mono">{manifestURL}</code>
            <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0 hover:bg-blue-100 dark:hover:bg-blue-900" onClick={copyURL}>
              {copied
                ? <Check className="h-4 w-4 text-emerald-500" />
                : <Copy className="h-4 w-4 text-blue-500" />}
            </Button>
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {cards.map((c) => (
          <Card key={c.title} className="overflow-hidden border-border/60 shadow-sm">
            <div className={`h-0.5 bg-gradient-to-r ${c.bar}`} />
            <CardHeader className="flex flex-row items-center justify-between pb-2 pt-4">
              <CardTitle className="text-xs font-medium text-muted-foreground uppercase tracking-wide">{c.title}</CardTitle>
              <div className={`h-8 w-8 rounded-lg ${c.iconBg} flex items-center justify-center`}>
                <c.icon className={`h-4 w-4 ${c.iconColor}`} />
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{c.value}</div>
              <p className="text-xs text-muted-foreground mt-1">{c.desc}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Download status */}
      <Card className="border-border/60 shadow-sm">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-medium">下载状态分布</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex gap-5 flex-wrap">
            {STATUS_BAR.map(({ label, key, color }) => (
              <div key={key} className="flex items-center gap-2">
                <div className={`w-2.5 h-2.5 rounded-full ${color}`} />
                <span className="text-sm text-muted-foreground">{label}</span>
                <span className="text-sm font-semibold tabular-nums">{byStatus[key] ?? 0}</span>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
