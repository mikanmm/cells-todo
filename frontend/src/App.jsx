import { useCallback, useEffect, useState } from 'react'
import { api } from './api.js'
import Backdrop from './Backdrop.jsx'
import Login from './Login.jsx'
import TodoApp from './TodoApp.jsx'

export default function App() {
  // undefined = checking the session; null = signed out
  const [user, setUser] = useState(undefined)

  useEffect(() => {
    api
      .me()
      .then(setUser)
      .catch(() => setUser(null))
  }, [])

  useEffect(() => {
    if (!user?.sessionExpiresAt) return
    let active = true
    let checking = false
    let timer

    async function checkSession() {
      if (checking) return
      checking = true
      window.clearTimeout(timer)
      try {
        const current = await api.me()
        if (active) setUser(current)
      } catch (error) {
        if (!active) return
        if (error.status === 401) setUser(null)
        else timer = window.setTimeout(checkSession, 30000)
      } finally {
        checking = false
      }
    }

    function onVisible() {
      if (document.visibilityState === 'visible') checkSession()
    }

    const remaining = Date.parse(user.sessionExpiresAt) - Date.now()
    timer = window.setTimeout(checkSession, Math.max(0, remaining) + 100)
    window.addEventListener('focus', checkSession)
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      active = false
      window.clearTimeout(timer)
      window.removeEventListener('focus', checkSession)
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [user])

  const handleLogout = useCallback(async () => {
    try {
      await api.logout()
    } finally {
      setUser(null)
    }
  }, [])

  useEffect(() => {
    if (user === undefined) return
    if (user === null && window.location.pathname !== '/login')
      window.history.replaceState(null, '', '/login')
    if (user && ['/login', '/'].includes(window.location.pathname))
      window.history.replaceState(null, '', '/dashboard')
  }, [user])

  return (
    <>
      <Backdrop />
      {user === undefined ? (
        <div className="boot" role="status">
          Loading…
        </div>
      ) : user === null ? (
        <Login onLogin={setUser} />
      ) : (
        <TodoApp user={user} onLogout={handleLogout} onSessionExpired={() => setUser(null)} />
      )}
    </>
  )
}
