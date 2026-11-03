import { useCallback, useEffect, useState } from 'react'
import { api } from './api.js'
import Backdrop from './Backdrop.jsx'
import Login from './Login.jsx'
import TodoApp from './TodoApp.jsx'

export default function App() {
  // undefined = sedang memeriksa sesi, null = belum masuk
  const [user, setUser] = useState(undefined)

  useEffect(() => {
    api
      .me()
      .then(setUser)
      .catch(() => setUser(null))
  }, [])

  const handleLogout = useCallback(async () => {
    try {
      await api.logout()
    } finally {
      setUser(null)
    }
  }, [])

  useEffect(() => {
    if (user === undefined) return
    if (user === null && window.location.pathname !== '/login') window.history.replaceState(null, '', '/login')
    if (user && ['/login', '/'].includes(window.location.pathname)) window.history.replaceState(null, '', '/dashboard')
  }, [user])

  return (
    <>
      <Backdrop />
      {user === undefined ? (
        <div className="boot" role="status">
          Memuat…
        </div>
      ) : user === null ? (
        <Login onLogin={setUser} />
      ) : (
        <TodoApp user={user} onLogout={handleLogout} onSessionExpired={() => setUser(null)} />
      )}
    </>
  )
}
