import { useEffect, useState } from 'react'
import { Sparkles } from 'lucide-react'
import { api, type Config } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { toast } from 'sonner'

export function Settings() {
  const [cfg, setCfg] = useState<Config | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    api.config.get().then(setCfg).finally(() => setLoading(false))
  }, [])

  const save = async () => {
    if (!cfg) return
    setSaving(true)
    try {
      await api.config.update(cfg)
      toast.success('配置已保存')
    } catch (e: unknown) {
      toast.error((e as Error).message)
    } finally {
      setSaving(false)
    }
  }

  const set = (path: string, value: unknown) => {
    setCfg((prev) => {
      if (!prev) return prev
      const next = structuredClone(prev)
      const keys = path.split('.')
      let obj: Record<string, unknown> = next as unknown as Record<string, unknown>
      for (let i = 0; i < keys.length - 1; i++) obj = obj[keys[i]] as Record<string, unknown>
      obj[keys[keys.length - 1]] = value
      return next
    })
  }

  if (loading) return <div className="p-8 text-center text-muted-foreground">加载中…</div>
  if (!cfg) return null

  return (
    <div className="p-6 space-y-6 max-w-2xl">
      <h1 className="text-2xl font-bold flex items-center gap-2">
        系统设置 <Sparkles className="h-5 w-5 text-sakura sparkle-pulse" />
      </h1>

      {/* 服务器 */}
      <Card>
        <CardHeader><CardTitle className="text-base">服务器</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1">
              <Label>监听地址</Label>
              <Input value={cfg.server.host} onChange={(e) => set('server.host', e.target.value)} />
            </div>
            <div className="space-y-1">
              <Label>端口</Label>
              <Input type="number" value={cfg.server.port} onChange={(e) => set('server.port', +e.target.value)} />
            </div>
          </div>
          <div className="space-y-1">
            <Label>公开访问地址 <span className="text-muted-foreground font-normal">（必填，用于生成 Manifest 下载链接）</span></Label>
            <Input
              value={cfg.server.public_url}
              onChange={(e) => set('server.public_url', e.target.value)}
              placeholder="https://your-domain.com:9443"
            />
            <p className="text-xs text-muted-foreground">
              Jellyfin 下载插件时使用此地址。若在反向代理后请务必填写外部 URL（含协议和端口），否则下载会失败。
            </p>
          </div>
        </CardContent>
      </Card>

      {/* 缓存 */}
      <Card>
        <CardHeader><CardTitle className="text-base">缓存策略</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1">
            <Label>Manifest TTL（秒）</Label>
            <Input type="number" value={cfg.cache.manifest_ttl_seconds}
              onChange={(e) => set('cache.manifest_ttl_seconds', +e.target.value)} />
            <p className="text-xs text-muted-foreground">默认 86400（24 小时）</p>
          </div>
          <div className="space-y-1">
            <Label>最大并发下载数</Label>
            <Input type="number" value={cfg.cache.max_concurrent_downloads}
              onChange={(e) => set('cache.max_concurrent_downloads', +e.target.value)} />
          </div>
        </CardContent>
      </Card>

      {/* 兼容性 */}
      <Card>
        <CardHeader><CardTitle className="text-base">插件兼容性</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1">
            <Label>你的 Jellyfin 服务器版本</Label>
            <Input
              value={cfg.compat.jellyfin_version}
              onChange={(e) => set('compat.jellyfin_version', e.target.value)}
              placeholder="10.11.11"
            />
            <p className="text-xs text-muted-foreground">
              填写后，插件目录会标注每个版本是否兼容——Jellyfin 允许安装任何 targetAbi 能解析的版本，
              但实际加载时会用服务器自身版本比对 targetAbi，不满足就显示"Not Supported"。这个设置只影响本面板的标注，不影响送给 Jellyfin 的数据。留空则不显示标注。
            </p>
          </div>
        </CardContent>
      </Card>

      {/* 存储 */}
      <Card>
        <CardHeader><CardTitle className="text-base">存储</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1">
            <Label>数据目录</Label>
            <Input value={cfg.storage.data_dir} onChange={(e) => set('storage.data_dir', e.target.value)} />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1">
              <Label>磁盘上限（MB）</Label>
              <Input type="number" value={cfg.storage.max_disk_mb}
                onChange={(e) => set('storage.max_disk_mb', +e.target.value)} />
            </div>
            <div className="space-y-1">
              <Label>每个插件保留版本数</Label>
              <Input type="number" value={cfg.storage.keep_versions}
                onChange={(e) => set('storage.keep_versions', +e.target.value)} />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 代理 */}
      <Card>
        <CardHeader><CardTitle className="text-base">上游代理</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1">
            <Label>代理类型</Label>
            <Select value={cfg.proxy.type || 'none'} onValueChange={(v) => set('proxy.type', v === 'none' ? '' : v)}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="none">不使用代理</SelectItem>
                <SelectItem value="http">HTTP</SelectItem>
                <SelectItem value="https">HTTPS</SelectItem>
                <SelectItem value="socks5">SOCKS5</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {cfg.proxy.type && (
            <>
              <div className="space-y-1">
                <Label>代理地址（host:port）</Label>
                <Input value={cfg.proxy.address} onChange={(e) => set('proxy.address', e.target.value)} placeholder="127.0.0.1:1080" />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <Label>用户名（可选）</Label>
                  <Input value={cfg.proxy.username} onChange={(e) => set('proxy.username', e.target.value)} />
                </div>
                <div className="space-y-1">
                  <Label>密码（可选）</Label>
                  <Input type="password" value={cfg.proxy.password} onChange={(e) => set('proxy.password', e.target.value)} />
                </div>
              </div>
              <div className="space-y-1">
                <Label>NO_PROXY（逗号分隔）</Label>
                <Input value={cfg.proxy.no_proxy} onChange={(e) => set('proxy.no_proxy', e.target.value)} placeholder="localhost,127.0.0.1" />
              </div>
            </>
          )}
        </CardContent>
      </Card>

      {/* 认证 */}
      <Card>
        <CardHeader><CardTitle className="text-base">管理页认证</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-2">
            <Switch checked={cfg.auth.enabled} onCheckedChange={(v) => set('auth.enabled', v)} />
            <Label>启用 HTTP Basic Auth</Label>
          </div>
          {cfg.auth.enabled && (
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1">
                <Label>用户名</Label>
                <Input value={cfg.auth.username} onChange={(e) => set('auth.username', e.target.value)} />
              </div>
              <div className="space-y-1">
                <Label>密码</Label>
                <Input type="password" value={cfg.auth.password} onChange={(e) => set('auth.password', e.target.value)} />
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <Separator />

      <div className="flex justify-end">
        <Button onClick={save} disabled={saving}>
          {saving ? '保存中…' : '保存配置'}
        </Button>
      </div>
    </div>
  )
}
