import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'

export default function CompleteTaskDialog({ todo, onConfirm, onCancel }) {
  const dialog = useRef(null)
  const saving = useRef(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    const element = dialog.current
    element.showModal()
    return () => element.close()
  }, [])

  async function confirm(e) {
    e.preventDefault()
    if (saving.current) return
    saving.current = true
    setBusy(true); setError('')
    try { await onConfirm() }
    catch (err) { setError(err.message) }
    finally { saving.current = false; setBusy(false) }
  }

  return createPortal(<dialog ref={dialog} className="complete-dialog" aria-labelledby="complete-title" aria-describedby="complete-description" onCancel={e => { e.preventDefault(); if (!saving.current) onCancel() }}>
    <form onSubmit={confirm}>
      <h2 id="complete-title">Selesaikan tugas?</h2>
      <p className="complete-task-name">{todo.title}</p>
      <p id="complete-description" className="muted">Tugas akan dipindahkan ke menu Selesai dan tidak muncul lagi di menu Semua.</p>
      {error && <p className="form-error" role="alert">{error}</p>}
      <div className="complete-actions">
        <button type="button" className="btn btn-ghost" disabled={busy} onClick={onCancel} autoFocus>Batal</button>
        <button type="submit" className="btn btn-primary" disabled={busy}>{busy ? 'Menyimpan...' : 'Ya, selesaikan'}</button>
      </div>
    </form>
  </dialog>, document.body)
}
