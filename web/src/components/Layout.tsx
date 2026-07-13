import { useEffect, useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { LayoutDashboard, Database, Package, Settings, ScrollText, Sparkles, BookOpen, LogOut, Sun, Moon } from 'lucide-react'
import { api, token } from '@/lib/api'
import { useTheme } from '@/hooks/use-theme'

const nav = [
  { to: '/', label: '仪表盘', icon: LayoutDashboard, end: true },
  { to: '/catalog', label: '插件目录', icon: BookOpen },
  { to: '/repos', label: '仓库管理', icon: Database },
  { to: '/packages', label: '插件包', icon: Package },
  { to: '/logs', label: '审计日志', icon: ScrollText },
  { to: '/settings', label: '系统设置', icon: Settings },
]

export function Layout() {
  const [version, setVersion] = useState<string>('—')
  const navigate = useNavigate()
  const { theme, toggle } = useTheme()

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
      <aside className="relative w-60 flex flex-col overflow-hidden border-r border-border/60 bg-card">
        {/* Decorative blobs */}
        <div className="pointer-events-none absolute -top-16 -left-16 h-40 w-40 rounded-full bg-sakura/20 blur-3xl float-slow" />
        <div className="pointer-events-none absolute bottom-24 -right-14 h-36 w-36 rounded-full bg-mint/20 blur-3xl float-slow" style={{ animationDelay: '2s' }} />

        {/* Brand */}
        <div className="relative flex items-center gap-3 px-4 py-5 border-b border-border/60">
          <div className="relative h-9 w-9 rounded-2xl bg-gradient-to-br from-sakura to-lavender flex items-center justify-center shadow-glow-sakura">
            <Sparkles className="h-4 w-4 text-white sparkle-pulse" />
          </div>
          <div>
            <p className="font-bold text-sm leading-none">Plugin Server</p>
            <p className="text-[10px] text-muted-foreground mt-1">Jellyfin ✧ 插件仓库</p>
          </div>
        </div>

        {/* Nav */}
        <nav className="relative flex-1 p-3 space-y-1 overflow-y-auto">
          {nav.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 rounded-full px-3.5 py-2 text-sm font-medium transition-all duration-200',
                  isActive
                    ? 'bg-gradient-to-r from-sakura/90 to-lavender/90 text-white shadow-glow-sakura'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                )
              }
            >
              {({ isActive }) => (
                <>
                  <Icon className={cn('h-4 w-4 shrink-0', isActive ? 'text-white' : '')} />
                  <span>{label}</span>
                </>
              )}
            </NavLink>
          ))}
        </nav>

        {/* Footer */}
        <div className="relative p-3 border-t border-border/60 space-y-2">
          <button
            onClick={toggle}
            className="w-full flex items-center gap-2.5 rounded-full px-3.5 py-2 text-xs font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground transition-all"
          >
            {theme === 'dark' ? <Moon className="h-3.5 w-3.5 shrink-0" /> : <Sun className="h-3.5 w-3.5 shrink-0" />}
            <span>{theme === 'dark' ? '夜间模式' : '日间模式'}</span>
          </button>
          <button
            onClick={logout}
            className="w-full flex items-center gap-2.5 rounded-full px-3.5 py-2 text-xs font-medium text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-all"
          >
            <LogOut className="h-3.5 w-3.5 shrink-0" />
            <span>退出登录</span>
          </button>
          <p className="text-[11px] text-muted-foreground/70 text-center font-mono pt-1">{version}</p>
        </div>
      </aside>

      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
