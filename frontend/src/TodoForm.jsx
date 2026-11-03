import { useId, useRef, useState } from 'react'
import { GemDot } from './Gem.jsx'

export const PRIORITIES = [
  { value: 'low', label: 'Rendah' },
  { value: 'medium', label: 'Sedang' },
  { value: 'high', label: 'Tinggi' },
]

const EMPTY = { title: '', notes: '', priority: 'medium', category: '', dueDate: '' }

// composer = true: kolom tambah cepat di atas daftar. false: form ubah di dalam kartu tugas.
export default function TodoForm({
  initial,
  categories = [],
  composer = false,
  submitLabel,
  onSubmit,
  onCancel,
}) {
  const uid = useId()
  const titleRef = useRef(null)
  const [values, setValues] = useState(initial ?? EMPTY)
  const [open, setOpen] = useState(!composer)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const set = (key) => (e) => {
    const value = e.target.value
    setValues((v) => ({ ...v, [key]: value }))
    if (key === 'title' && value.trim()) {
      setError((current) => (current === 'Tulis judul tugas dulu.' ? '' : current))
    }
  }

  async function submit(e) {
    e.preventDefault()
    if (busy) return
    if (!values.title.trim()) {
      setError('Tulis judul tugas dulu.')
      titleRef.current?.focus()
      return
    }
    setBusy(true)
    setError('')
    try {
      await onSubmit({ ...values, title: values.title.trim() })
      if (composer) {
        setValues(EMPTY)
        setOpen(false)
        titleRef.current?.blur()
      }
    } catch (err) {
      setError(err.message)
    } finally {
      setBusy(false)
    }
  }

  function cancel() {
    if (busy) return
    setError('')
    if (composer) {
      setValues(EMPTY)
      setOpen(false)
      titleRef.current?.blur()
    } else onCancel?.()
  }

  function onKeyDown(e) {
    if (e.key === 'Escape') {
      e.preventDefault()
      cancel()
    }
  }

  return (
    <form
      className={`card form ${composer ? 'composer' : 'editor'}`}
      onSubmit={submit}
      onKeyDown={onKeyDown}
      noValidate
    >
      <div className="form-main">
        <input
          ref={titleRef}
          className="title-input"
          type="text"
          placeholder={composer ? 'Tulis tugas baru…' : 'Judul tugas'}
          aria-label="Judul tugas"
          maxLength={200}
          value={values.title}
          onChange={set('title')}
          onFocus={() => setOpen(true)}
          autoFocus={!composer}
        />
        {composer && (
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? 'Menyimpan…' : submitLabel}
          </button>
        )}
      </div>

      {open && (
        <div className="form-details">
          <label className="field field-wide">
            <span>Catatan</span>
            <textarea rows={2} maxLength={2000} value={values.notes} onChange={set('notes')} />
          </label>

          <fieldset className="field">
            <legend>Prioritas</legend>
            <div className="segmented">
              {PRIORITIES.map((p) => (
                <label key={p.value} className={`seg ${values.priority === p.value ? 'on' : ''}`}>
                  <input
                    type="radio"
                    name={`${uid}-priority`}
                    value={p.value}
                    checked={values.priority === p.value}
                    onChange={set('priority')}
                    className="visually-hidden"
                  />
                  <GemDot tone={p.value} />
                  {p.label}
                </label>
              ))}
            </div>
          </fieldset>

          <label className="field">
            <span>Tenggat</span>
            <input type="date" value={values.dueDate} onChange={set('dueDate')} />
          </label>

          <label className="field">
            <span>Kategori</span>
            <select value={values.category} onChange={set('category')}>
              <option value="">Tanpa kategori</option>
              {categories.map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
          </label>
        </div>
      )}

      {error && (
        <p className="form-error" role="alert">
          {error}
        </p>
      )}

      {(!composer || open) && (
        <div className="form-actions">
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? 'Menyimpan…' : submitLabel}
          </button>
          <button type="button" className="btn btn-ghost" disabled={busy} onClick={cancel}>
            Batal
          </button>
        </div>
      )}
    </form>
  )
}
