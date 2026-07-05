import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, token } from '@/lib/api'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Loader2, Lock, User, Eye, EyeOff, Sparkles } from 'lucide-react'

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
    <div className="min-h-screen flex items-center justify-center bg-background px-4 relative overflow-hidden">
      {/* Background decoration — soft floating blobs */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute -top-32 -right-32 w-96 h-96 bg-sakura/25 rounded-full blur-3xl float-slow" />
        <div className="absolute -bottom-32 -left-32 w-96 h-96 bg-lavender/25 rounded-full blur-3xl float-slow" style={{ animationDelay: '2.5s' }} />
        <div className="absolute top-1/3 left-1/4 w-64 h-64 bg-mint/15 rounded-full blur-3xl float-slow" style={{ animationDelay: '1.2s' }} />
      </div>

      <div className="relative w-full max-w-sm">
        {/* Card */}
        <div className="bg-card/90 backdrop-blur-xl border border-border/60 rounded-2xl p-8 shadow-glow-sakura">
          {/* Logo */}
          <div className="flex flex-col items-center mb-8">
            <div className="relative h-16 w-16 rounded-2xl bg-gradient-to-br from-sakura to-lavender flex items-center justify-center shadow-glow-sakura mb-4">
              <Sparkles className="h-8 w-8 text-white sparkle-pulse" />
            </div>
            <h1 className="text-xl font-bold tracking-tight">Jellyfin Plugin Server</h1>
            <p className="text-sm text-muted-foreground mt-1">欢迎回来，请登录以继续 ✧</p>
          </div>

          {/* Form */}
          <form onSubmit={submit} className="space-y-4">
            <div className="relative">
              <User className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                type="text"
                placeholder="用户名"
                value={username}
                onChange={e => setUsername(e.target.value)}
                autoComplete="username"
                autoFocus
                required
                className="pl-9 h-11"
              />
            </div>

            <div className="relative">
              <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                type={showPwd ? 'text' : 'password'}
                placeholder="密码"
                value={password}
                onChange={e => setPassword(e.target.value)}
                autoComplete="current-password"
                required
                className="pl-9 pr-10 h-11"
              />
              <button
                type="button"
                tabIndex={-1}
                onClick={() => setShowPwd(v => !v)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
              >
                {showPwd ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>

            {error && (
              <div className="text-sm text-destructive bg-destructive/10 border border-destructive/20 rounded-xl px-3 py-2">
                {error}
              </div>
            )}

            <Button type="submit" disabled={loading} className="w-full h-11 font-medium">
              {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : '登 录'}
            </Button>
          </form>
        </div>

        <p className="text-center text-muted-foreground/60 text-xs mt-6">
          Jellyfin Plugin Server · 自托管插件仓库
        </p>
      </div>
    </div>
  )
}
