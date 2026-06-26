import { useEffect, useState } from 'react'
import { api, type LogEntry } from '@/lib/api'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { RefreshCw } from 'lucide-react'

const LEVEL_VARIANT: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  INFO: 'secondary',
  WARN: 'outline',
  ERROR: 'destructive',
  DEBUG: 'outline',
}

export function Logs() {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [loading, setLoading] = useState(true)

  const load = () => {
    setLoading(true)
    api.logs.list().then((l) => setLogs(l ?? [])).finally(() => setLoading(false))
  }

  useEffect(load, [])

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">操作日志</h1>
        <Button variant="outline" size="sm" onClick={load}>
          <RefreshCw className={`h-4 w-4 mr-1 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </Button>
      </div>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-8 text-center text-muted-foreground">加载中…</div>
          ) : logs.length === 0 ? (
            <div className="p-8 text-center text-muted-foreground">暂无日志</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-28">级别</TableHead>
                  <TableHead>消息</TableHead>
                  <TableHead>详情</TableHead>
                  <TableHead className="w-40">时间</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {logs.map((l) => (
                  <TableRow key={l.id}>
                    <TableCell>
                      <Badge variant={LEVEL_VARIANT[l.level] ?? 'outline'}>{l.level}</Badge>
                    </TableCell>
                    <TableCell className="font-medium text-sm">{l.message}</TableCell>
                    <TableCell className="text-xs text-muted-foreground max-w-xs truncate">{l.detail || '—'}</TableCell>
                    <TableCell className="text-xs text-muted-foreground whitespace-nowrap">
                      {new Date(l.created_at).toLocaleString('zh-CN')}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
