import { useState } from 'react'
import { api } from './api.js'

export default function CategorySettings({ categories, onChange, onSessionExpired }) {
  const [name, setName] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  function fail(err) {
    if (err.status === 401) onSessionExpired()
    else setError(err.message)
  }

  async function add(e) {
    e.preventDefault()
    if (busy) return
    const trimmed = name.trim()
    setError('')
    setNotice('')
    if (!trimmed) {
      setError('Please enter a category name.')
      return
    }
    setBusy(true)
    try {
      await api.addCategory(trimmed)
      setName('')
      await onChange()
      setNotice('Category added successfully.')
    } catch (err) {
      fail(err)
    } finally {
      setBusy(false)
    }
  }

  async function remove(category) {
    if (busy) return
    setBusy(true)
    setError('')
    setNotice('')
    try {
      await api.deleteCategory(category)
      await onChange()
      setNotice('Category deleted successfully.')
    } catch (err) {
      fail(err)
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="card settings-panel">
      <h2>Categories</h2>
      <form className="category-create" onSubmit={add} noValidate>
        <label className="field" htmlFor="category-name">
          <span>Category name</span>
          <input
            id="category-name"
            value={name}
            maxLength={40}
            disabled={busy}
            onChange={(e) => setName(e.target.value)}
            placeholder="For example: Work"
          />
        </label>
        <button type="submit" className="btn btn-primary" disabled={busy}>
          {busy ? 'Saving...' : 'Add category'}
        </button>
      </form>
      {error && (
        <p className="form-error" role="alert">
          {error}
        </p>
      )}
      {notice && (
        <p className="category-notice" role="status">
          {notice}
        </p>
      )}
      <ul className="category-settings-list">
        {categories.map((category) => (
          <li key={category}>
            <span>{category}</span>
            <button
              type="button"
              className="link-btn danger"
              disabled={busy}
              onClick={() => remove(category)}
            >
              Delete
            </button>
          </li>
        ))}
      </ul>
      {!categories.length && (
        <p className="muted">No categories yet. Add your first category above.</p>
      )}
      <p className="muted settings-help">Deleting a category leaves its tasks uncategorized.</p>
    </section>
  )
}
