import { BrowserRouter, Routes, Route, useLocation, useNavigate } from 'react-router-dom'
import { useEffect, useState } from 'react'
import TargetsPage from './pages/TargetsPage'
import DashboardPage from './pages/DashboardPage'
import LoginPage from './pages/LoginPage'

function AuthGate({ children }: { children: React.ReactNode }) {
  const [authed, setAuthed] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    if (location.pathname === '/login') {
      setAuthed(true)
      return
    }
    fetch('/api/targets').then(res => {
      if (res.status === 401) navigate('/login', { replace: true })
      else setAuthed(true)
    }).catch(() => navigate('/login', { replace: true }))
  }, [location.pathname, navigate])

  if (!authed) {
    return (
      <div style={{
        position: 'fixed', inset: 0, background: 'var(--bg)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        zIndex: 9999,
      }}>
        <div className="spinner" style={{ width: 24, height: 24, borderWidth: 3 }} />
      </div>
    )
  }

  return <>{children}</>
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthGate>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/" element={<TargetsPage />} />
          <Route path="/dashboard" element={<DashboardPage />} />
        </Routes>
      </AuthGate>
    </BrowserRouter>
  )
}
