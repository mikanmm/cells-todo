import { useState } from 'react'
import { api } from './api.js'
import { GemStatic } from './Gem.jsx'

export default function Login({ onLogin }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e) {
    e.preventDefault()
    if (busy) return
    setBusy(true)
    setError('')
    try {
      const user = await api.login(username.trim(), password)
      onLogin(user)
    } catch (err) {
      setError(err.message)
      setPassword('')
    } finally {
      setBusy(false)
    }
  }

  return (
    <main className="login-wrap">
      <section className="card login-card">
        <div className="login-gems" aria-hidden="true">
          <GemStatic tone="low" size={34} />
          <GemStatic tone="medium" size={56} />
          <GemStatic tone="high" size={34} />
        </div>
        <h1 className="brand-title">cells</h1>
        <p className="muted">Masuk untuk melihat daftar tugas Anda.</p>

        <form onSubmit={submit} className="login-form">
          <label className="field">
            <span>Username</span>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="username"
              autoCapitalize="none"
              spellCheck={false}
              autoFocus
              required
            />
          </label>
          <label className="field">
            <span>Password</span>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
              required
            />
          </label>

          {error && (
            <p className="form-error" role="alert">
              {error}
            </p>
          )}

          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? 'Memeriksa…' : 'Masuk'}
          </button>
        </form>

        <p className="login-note">Akun dibuat oleh admin. Belum punya akun? Minta admin membuatkannya.</p>
      </section>
    </main>
  )
}
