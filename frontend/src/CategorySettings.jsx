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
      setError('Isi nama kategori terlebih dahulu.')
      return
    }
    setBusy(true)
    try {
      await api.addCategory(trimmed)
      setName('')
      await onChange()
      setNotice('Kategori berhasil ditambahkan.')
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
      setNotice('Kategori berhasil dihapus.')
    } catch (err) {
      fail(err)
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="card settings-panel">
      <h2>List kategori</h2>
      <form className="category-create" onSubmit={add} noValidate>
        <label className="field" htmlFor="category-name">
          <span>Nama kategori</span>
          <input
            id="category-name"
            value={name}
            maxLength={40}
            disabled={busy}
            onChange={(e) => setName(e.target.value)}
            placeholder="Cth. kategori: Kerja"
          />
        </label>
        <button type="submit" className="btn btn-primary" disabled={busy}>
          {busy ? 'Menyimpan...' : 'Tambah kategori'}
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
              Hapus
            </button>
          </li>
        ))}
      </ul>
      {!categories.length && (
        <p className="muted">Belum ada kategori. Tambahkan kategori pertama di atas.</p>
      )}
      <p className="muted settings-help">
        Menghapus kategori membuat tugas terkait menjadi tanpa kategori.
      </p>
    </section>
  )
}
