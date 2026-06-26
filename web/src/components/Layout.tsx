import { useEffect, useState } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { LayoutDashboard, Database, Package, Settings, ScrollText, Tv } from 'lucide-react'
import { api } from '@/lib/api'

const nav = [
  { to: '/', label: '仪表盘', icon: LayoutDashboard, end: true },
  { to: '/repos', label: '仓库管理', icon: Database },
  { to: '/packages', label: '插件包', icon: Package },
  { to: '/logs', label: '操作日志', icon: ScrollText },
  { to: '/settings', label: '系统设置', icon: Settings },
]

export function Layout() {
  const [version, setVersion] = useState<string>('—')

  useEffect(() => {
    api.status().then((s) => setVersion(s.version)).catch(() => {})
  }, [])

  return (
    <div className="flex h-screen bg-background">
      <aside className="w-56 border-r flex flex-col">
        <div className="flex items-center gap-2 px-4 py-4 border-b">
          <Tv className="h-5 w-5 text-primary" />
          <span className="font-semibold text-sm">Jellyfin Plugin Server</span>
        </div>
        <nav className="flex-1 p-2 space-y-1">
          {nav.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors',
                  isActive
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                )
              }
            >
              <Icon className="h-4 w-4" />
              {label}
            </NavLink>
          ))}
        </nav>
        <div className="p-3 border-t">
          <p className="text-xs text-muted-foreground text-center">{version}</p>
        </div>
      </aside>

      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
