import { useId, useRef, useState } from 'react'
import { GemDot } from './Gem.jsx'

export const PRIORITIES = [
  { value: 'low', label: 'Low' },
  { value: 'medium', label: 'Medium' },
  { value: 'high', label: 'High' },
]

export const REPEAT_DAYS = [
  'Monday',
  'Tuesday',
  'Wednesday',
  'Thursday',
  'Friday',
  'Saturday',
  'Sunday',
]
const EMPTY = {
  title: '',
  notes: '',
  priority: 'medium',
  category: '',
  dueDate: '',
  repeatType: 'none',
  repeatDays: [],
  notify: false,
}

// composer = true: quick-entry form above the list; false: edit form inside a task card.
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
  const [values, setValues] = useState({ ...EMPTY, ...initial })
  const [open, setOpen] = useState(!composer)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const set = (key) => (e) => {
    const value = e.target.value
    setValues((v) => ({ ...v, [key]: value }))
    if (key === 'title' && value.trim()) {
      setError((current) => (current === 'Please enter a task title.' ? '' : current))
    }
  }

  async function submit(e) {
    e.preventDefault()
    if (busy) return
    if (!values.title.trim()) {
      setError('Please enter a task title.')
      titleRef.current?.focus()
      return
    }
    if (values.repeatType === 'custom' && values.repeatDays.length === 0) {
      setError('Select at least one repeat day.')
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
          placeholder={composer ? 'Enter a new task…' : 'Task title'}
          aria-label="Task title"
          maxLength={200}
          value={values.title}
          onChange={set('title')}
          onFocus={() => setOpen(true)}
          autoFocus={!composer}
        />
        {composer && (
          <button type="submit" className="btn btn-primary" disabled={busy}>
            {busy ? 'Saving…' : submitLabel}
          </button>
        )}
      </div>

      {open && (
        <div className="form-details">
          <label className="field field-wide">
            <span>Notes</span>
            <textarea rows={2} maxLength={2000} value={values.notes} onChange={set('notes')} />
          </label>

          <fieldset className="field">
            <legend>Priority</legend>
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
              <label className="notify-toggle">
                <span>Notify</span>
                <input
                  type="checkbox"
                  role="switch"
                  checked={values.notify}
                  onChange={(e) => setValues((v) => ({ ...v, notify: e.target.checked }))}
                />
                <span className="toggle-track" aria-hidden="true" />
              </label>
              <small className="muted">Include this task in email and Discord notifications.</small>
            </div>
          </fieldset>

          <div className="field due-repeat">
            <label className="field">
              <span>Due date</span>
              <input type="date" value={values.dueDate} onChange={set('dueDate')} />
            </label>
            <label className="field">
              <span>Repeat</span>
              <select
                value={values.repeatType}
                onChange={(e) => {
                  const repeatType = e.target.value
                  setValues((v) => ({
                    ...v,
                    repeatType,
                    repeatDays:
                      repeatType === 'weekdays'
                        ? [1, 2, 3, 4, 5]
                        : repeatType === 'daily'
                          ? [1, 2, 3, 4, 5, 6, 7]
                          : [],
                  }))
                }}
              >
                <option value="none">Does not repeat</option>
                <option value="weekdays">5 Days (Monday–Friday)</option>
                <option value="daily">7 Days (Monday–Sunday)</option>
                <option value="custom">Custom days</option>
              </select>
            </label>
            {values.repeatType !== 'none' && (
              <fieldset className="field">
                <legend>Repeat days</legend>
                <div className="repeat-days">
                  {REPEAT_DAYS.map((day, i) => (
                    <label
                      key={day}
                      className={`repeat-day ${values.repeatDays.includes(i + 1) ? 'on' : ''}`}
                    >
                      <input
                        type="checkbox"
                        className="visually-hidden"
                        aria-label={day}
                        disabled={values.repeatType !== 'custom'}
                        checked={values.repeatDays.includes(i + 1)}
                        onChange={(e) =>
                          setValues((v) => ({
                            ...v,
                            repeatDays: e.target.checked
                              ? [...v.repeatDays, i + 1].sort((a, b) => a - b)
                              : v.repeatDays.filter((d) => d !== i + 1),
                          }))
                        }
                      />
                      {day.slice(0, 3)}
                    </label>
                  ))}
                </div>
              </fieldset>
            )}
            {values.repeatType !== 'none' && (
              <small className="muted">
                Completed tasks reopen on the next selected day (Asia/Jakarta).
              </small>
            )}
          </div>

          <label className="field">
            <span>Category</span>
            <select value={values.category} onChange={set('category')}>
              <option value="">Uncategorized</option>
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
            {busy ? 'Saving…' : submitLabel}
          </button>
          <button type="button" className="btn btn-ghost" disabled={busy} onClick={cancel}>
            Cancel
          </button>
        </div>
      )}
    </form>
  )
}
