import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, token } from '@/lib/api'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Loader2, Lock, User, Eye, EyeOff, Server } from 'lucide-react'

export function Login() {
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [showPwd, setShowPwd] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // Already authenticated → skip login
  useEffect(() => {
    if (token.get()) navigate('/', { replace: true })
  }, [navigate])

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const { token: t } = await api.auth.login(username, password)
      token.set(t)
      navigate('/', { replace: true })
    } catch (err: unknown) {
      setError((err as Error).message || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-950 via-blue-950 to-slate-900 px-4">
      {/* Background decoration */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute -top-40 -right-40 w-96 h-96 bg-blue-500/10 rounded-full blur-3xl" />
        <div className="absolute -bottom-40 -left-40 w-96 h-96 bg-violet-500/10 rounded-full blur-3xl" />
      </div>

      <div className="relative w-full max-w-sm">
        {/* Card */}
        <div className="bg-white/5 backdrop-blur-xl border border-white/10 rounded-2xl p-8 shadow-2xl">
          {/* Logo */}
          <div className="flex flex-col items-center mb-8">
            <div className="h-14 w-14 rounded-2xl bg-gradient-to-br from-blue-500 to-violet-600 flex items-center justify-center shadow-lg shadow-blue-500/25 mb-4">
              <Server className="h-7 w-7 text-white" />
            </div>
            <h1 className="text-xl font-bold text-white tracking-tight">Jellyfin Plugin Server</h1>
            <p className="text-sm text-white/40 mt-1">请登录以继续</p>
          </div>

          {/* Form */}
          <form onSubmit={submit} className="space-y-4">
            <div className="relative">
              <User className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-white/30" />
              <Input
                type="text"
                placeholder="用户名"
                value={username}
                onChange={e => setUsername(e.target.value)}
                autoComplete="username"
                autoFocus
                required
                className="pl-9 bg-white/5 border-white/10 text-white placeholder:text-white/30 focus:border-blue-500/60 focus:ring-blue-500/20 h-11"
              />
            </div>

            <div className="relative">
              <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-white/30" />
              <Input
                type={showPwd ? 'text' : 'password'}
                placeholder="密码"
                value={password}
                onChange={e => setPassword(e.target.value)}
                autoComplete="current-password"
                required
                className="pl-9 pr-10 bg-white/5 border-white/10 text-white placeholder:text-white/30 focus:border-blue-500/60 focus:ring-blue-500/20 h-11"
              />
              <button
                type="button"
                tabIndex={-1}
                onClick={() => setShowPwd(v => !v)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-white/30 hover:text-white/60 transition-colors"
              >
                {showPwd ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>

            {error && (
              <div className="text-sm text-red-400 bg-red-400/10 border border-red-400/20 rounded-lg px-3 py-2">
                {error}
              </div>
            )}

            <Button
              type="submit"
              disabled={loading}
              className="w-full h-11 bg-gradient-to-r from-blue-500 to-violet-600 hover:from-blue-600 hover:to-violet-700 border-0 text-white font-medium shadow-lg shadow-blue-500/20 transition-all"
            >
              {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : '登 录'}
            </Button>
          </form>
        </div>

        <p className="text-center text-white/20 text-xs mt-6">
          Jellyfin Plugin Server · 自托管插件仓库
        </p>
      </div>
    </div>
  )
}
