import { useEffect, useState } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from '@/components/Layout'
import { Dashboard } from '@/pages/Dashboard'
import { Catalog } from '@/pages/Catalog'
import { Repos } from '@/pages/Repos'
import { Packages } from '@/pages/Packages'
import { Logs } from '@/pages/Logs'
import { Settings } from '@/pages/Settings'
import { Login } from '@/pages/Login'
import { Toaster } from '@/components/ui/sonner'
import { api, token } from '@/lib/api'

function ProtectedLayout() {
  const [ready, setReady] = useState(false)
  const [needsLogin, setNeedsLogin] = useState(false)

  useEffect(() => {
    api.auth.status().then(({ enabled }) => {
      if (!enabled) {
        // Auth disabled — issue a dummy token so API calls succeed
        if (!token.get()) {
          api.auth.login('', '').then(({ token: t }) => token.set(t)).catch(() => {})
        }
        setReady(true)
      } else if (!token.get()) {
        setNeedsLogin(true)
        setReady(true)
      } else {
        setReady(true)
      }
    }).catch(() => setReady(true))
  }, [])

  if (!ready) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="h-8 w-8 rounded-full border-2 border-primary border-t-transparent animate-spin" />
      </div>
    )
  }

  if (needsLogin) {
    return <Navigate to="/login" replace />
  }

  return <Layout />
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route element={<ProtectedLayout />}>
          <Route index element={<Dashboard />} />
          <Route path="catalog" element={<Catalog />} />
          <Route path="repos" element={<Repos />} />
          <Route path="packages" element={<Packages />} />
          <Route path="logs" element={<Logs />} />
          <Route path="settings" element={<Settings />} />
        </Route>
      </Routes>
      <Toaster richColors position="top-right" />
    </BrowserRouter>
  )
}
