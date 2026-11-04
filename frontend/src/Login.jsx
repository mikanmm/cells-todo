import { useState } from 'react'
import { api } from './api.js'
import { GemStatic } from './Gem.jsx'

export default function Login({ onLogin }) {
  const [registering, setRegistering] = useState(false)
  const [displayName, setDisplayName] = useState('')
  const [email, setEmail] = useState('')
  const [code, setCode] = useState('')
  const [sent, setSent] = useState(false)
  const [success, setSuccess] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function submit(e) {
    e.preventDefault()
    if (busy) return
    setBusy(true)
    setError('')
    setSuccess('')
    try {
      if (registering) {
        await api.register(username.trim(), password, displayName.trim(), email, sent ? code : '')
        if (!sent) {
          setSent(true)
          setSuccess('Check your email for the six-digit code. It expires in 10 minutes.')
          return
        }
        setSent(false)
        setCode('')
        setRegistering(false)
        setPassword('')
        setSuccess('Account created. Sign in to start your own task list.')
        return
      }
      const user = await api.login(username.trim(), password)
      onLogin(user)
    } catch (err) {
      setError(err.message)
      if (!registering) setPassword('')
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
        <h1 className="brand-title">Cells</h1>
        <p className="muted">{registering ? 'Create your account for your own task list.' : 'Sign in to view your tasks.'}</p>

        <form onSubmit={submit} className="login-form">
          {registering && <label className="field"><span>Display name</span><input value={displayName} onChange={(e) => setDisplayName(e.target.value)} maxLength={100} autoComplete="nickname" /></label>}
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
              autoComplete={registering ? "new-password" : "current-password"}
              minLength={registering ? 8 : undefined}
              required
            />
          </label>

          {error && (
            <p className="form-error" role="alert">
              {error}
            </p>
          )}

          {registering && <label className="field"><span>Email</span>
            <input type="email" autoComplete="email" value={email} required disabled={sent || busy} onChange={e => setEmail(e.target.value)} />
          </label>}
          {registering && sent && <>
            <label className="field"><span>Email verification code</span>
              <input autoComplete="one-time-code" inputMode="numeric" pattern="[0-9]{6}" maxLength={6} value={code} required onChange={e => setCode(e.target.value)} />
            </label>
            <button type="button" className="login-link" disabled={busy} onClick={() => { setSent(false); setCode(''); setSuccess(''); setError('') }}>Change email or request another code</button>
          </>}
          <button type="button" className="login-link" disabled={busy} onClick={() => {
            setRegistering(!registering)
            setSent(false)
            setCode('')
            setError('')
            setSuccess('')
            setPassword('')
          }}>{registering ? 'Back to sign in' : 'Create an account'}</button>
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? 'Checking…' : registering ? (sent ? 'Verify and create account' : 'Send verification code') : 'Sign in'}
          </button>
        </form>

        {success && <p role="status">{success}</p>}

      </section>
    </main>
  )
}
