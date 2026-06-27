import { useEffect, useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { LayoutDashboard, Database, Package, Settings, ScrollText, Tv, BookOpen, LogOut } from 'lucide-react'
import { api, token } from '@/lib/api'

const nav = [
  { to: '/', label: '仪表盘', icon: LayoutDashboard, end: true },
  { to: '/catalog', label: '插件目录', icon: BookOpen },
  { to: '/repos', label: '仓库管理', icon: Database },
  { to: '/packages', label: '插件包', icon: Package },
  { to: '/logs', label: '操作日志', icon: ScrollText },
  { to: '/settings', label: '系统设置', icon: Settings },
]

export function Layout() {
  const [version, setVersion] = useState<string>('—')
  const navigate = useNavigate()

  useEffect(() => {
    api.status().then((s) => setVersion(s.version)).catch(() => {})
  }, [])

  const logout = async () => {
    await api.auth.logout().catch(() => {})
    token.clear()
    navigate('/login', { replace: true })
  }

  return (
    <div className="flex h-screen bg-background">
      <aside className="w-56 flex flex-col bg-slate-900 border-r border-slate-800">
        {/* Brand */}
        <div className="flex items-center gap-3 px-4 py-5 border-b border-slate-800">
          <div className="h-8 w-8 rounded-xl bg-gradient-to-br from-blue-500 to-violet-600 flex items-center justify-center shadow-lg shadow-blue-900/40">
            <Tv className="h-4 w-4 text-white" />
          </div>
          <div>
            <p className="font-semibold text-sm text-white leading-none">Plugin Server</p>
            <p className="text-[10px] text-slate-500 mt-0.5">Jellyfin</p>
          </div>
        </div>

        {/* Nav */}
        <nav className="flex-1 p-2 space-y-0.5 overflow-y-auto">
          {nav.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-all duration-150 border',
                  isActive
                    ? 'bg-blue-500/15 text-blue-400 border-blue-500/30 shadow-sm'
                    : 'text-slate-400 hover:bg-slate-800 hover:text-slate-100 border-transparent'
                )
              }
            >
              {({ isActive }) => (
                <>
                  <Icon className={cn('h-4 w-4 shrink-0', isActive ? 'text-blue-400' : '')} />
                  <span>{label}</span>
                </>
              )}
            </NavLink>
          ))}
        </nav>

        {/* Footer */}
        <div className="p-3 border-t border-slate-800 space-y-2">
          <button
            onClick={logout}
            className="w-full flex items-center gap-2.5 rounded-lg px-3 py-2 text-xs text-slate-500 hover:bg-slate-800 hover:text-red-400 transition-all border border-transparent"
          >
            <LogOut className="h-3.5 w-3.5 shrink-0" />
            <span>退出登录</span>
          </button>
          <p className="text-[11px] text-slate-600 text-center font-mono">{version}</p>
        </div>
      </aside>

      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
