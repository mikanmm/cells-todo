import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { api } from './api.js'
import NotifyHourPicker from './NotifyHourPicker.jsx'

function DiscordSuccess({ message, onClose }) {
  const dialog = useRef(null)
  useEffect(() => {
    const element = dialog.current
    element.showModal()
    return () => element.close()
  }, [])
  return createPortal(
    <dialog
      ref={dialog}
      className="complete-dialog"
      aria-labelledby="discord-success-title"
      onCancel={(e) => {
        e.preventDefault()
        onClose()
      }}
    >
      <div className="mail-success-content">
        <span className="mail-success-icon" aria-hidden="true">
          ✓
        </span>
        <h2 id="discord-success-title">Message sent successfully</h2>
        <p>{message}</p>
        <button className="btn btn-primary" type="button" autoFocus onClick={onClose}>
          OK
        </button>
      </div>
    </dialog>,
    document.body,
  )
}

export default function DiscordNotifySettings({ onSessionExpired }) {
  const [info, setInfo] = useState(null)
  const [config, setConfig] = useState(null)
  const [webhook, setWebhook] = useState('')
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [preview, setPreview] = useState('')
  const [success, setSuccess] = useState('')
  const [reload, setReload] = useState(0)
  function fail(err) {
    if (err.status === 401) onSessionExpired()
    else setError(err.message)
  }
  useEffect(() => {
    let active = true
    api
      .discordSettings()
      .then((data) => {
        if (active) {
          setInfo(data)
          setConfig(data.schedule)
        }
      })
      .catch((err) => {
        if (active) fail(err)
      })
    return () => {
      active = false
    }
  }, [reload])
  function change(key, value) {
    setConfig((current) => ({ ...current, [key]: value }))
    setPreview('')
  }
  async function action(kind) {
    if (busy) return
    setBusy(kind)
    setError('')
    setNotice('')
    try {
      if (kind === 'save') {
        const data = await api.saveDiscord({ ...config, webhook })
        setInfo(data)
        setConfig(data.schedule)
        setWebhook('')
        setNotice('Discord webhook, template, and schedule saved successfully.')
      } else if (kind === 'preview') {
        setPreview((await api.previewDiscord(config)).content)
      } else {
        const data = await api.sendDiscord({ ...config, test: kind === 'test' })
        setSuccess(data.message)
      }
    } catch (err) {
      fail(err)
    } finally {
      setBusy('')
    }
  }
  if (!config)
    return (
      <section className="settings-panel">
        {error ? (
          <>
            <p role="alert" className="form-error">
              {error}
            </p>
            <button
              type="button"
              className="btn btn-ghost"
              onClick={() => {
                setError('')
                setReload((value) => value + 1)
              }}
            >
              Reload
            </button>
          </>
        ) : (
          <p role="status">Loading Discord settings...</p>
        )}
      </section>
    )
  return (
    <section className="settings-panel discord-panel" aria-busy={!!busy}>
      <div>
        <h2>Discord Webhook</h2>
        <p className="muted">Send reminders to a Discord text channel using a webhook.</p>
      </div>
      {error && (
        <p role="alert" className="form-error">
          {error}
        </p>
      )}
      {notice && (
        <p role="status" className="category-notice">
          {notice}
        </p>
      )}
      <p className={`notify-status ${info.configured ? 'ready' : ''}`}>
        {info.configured ? 'Webhook saved' : 'Webhook not configured'}
      </p>
      <form
        onSubmit={(e) => {
          e.preventDefault()
          action('save')
        }}
      >
        <label className="field" htmlFor="discord-webhook">
          <span>Discord webhook URL</span>
          <input
            id="discord-webhook"
            type="password"
            autoComplete="new-password"
            value={webhook}
            maxLength={512}
            disabled={!!busy}
            onChange={(e) => setWebhook(e.target.value)}
            placeholder={
              info.configured
                ? 'Leave blank to keep the saved webhook'
                : 'https://discord.com/api/webhooks/...'
            }
          />
        </label>
        <div className="notify-schedule">
          <h3>Daily schedule</h3>
          <label>
            <input
              type="checkbox"
              checked={config.enabled}
              disabled={!!busy}
              onChange={(e) => change('enabled', e.target.checked)}
            />{' '}
            Enable automatic delivery to Discord
          </label>
          <span>Delivery time (WIB)</span>
          <NotifyHourPicker
            value={config.hour}
            disabled={!!busy}
            onChange={(value) => change('hour', value)}
          />
          <p className="muted settings-help">
            Runs daily in the Asia/Jakarta time zone. The server must remain running. A missed
            delivery is processed once when the server becomes available again.
          </p>
          {info.nextRun?.Valid && (
            <p className="muted">
              Next scheduled delivery:{' '}
              {new Date(info.nextRun.Time).toLocaleString('en-GB', { timeZone: 'Asia/Jakarta' })}{' '}
              WIB
            </p>
          )}
          {info.lastStatus && <p className="muted">Last delivery status: {info.lastStatus}</p>}
        </div>

        <p className="muted settings-help">
          Create a webhook in Discord channel settings → Integrations → Webhooks. The URL is stored
          on the server and is not displayed again. Save URL changes before sending a test.
        </p>
        <label className="field" htmlFor="discord-mode">
          <span>Message mode</span>
          <select
            id="discord-mode"
            value={config.mode}
            disabled={!!busy}
            onChange={(e) => change('mode', e.target.value)}
          >
            <option value="text">Plain text</option>
            <option value="template">Custom template (Go template)</option>
          </select>
        </label>
        {config.mode === 'template' && (
          <div className="notify-hint">
            <p>Use the following fields from active tasks with Notify enabled:</p>
            <code>{'{{.Name}} {{.Date}} {{.Total}} {{.Urgent}} {{.Medium}} {{.Low}}'}</code>
            <p>
              <code>{'{{range .HighTasks}}- {{.Title}}{{end}}'}</code>
            </p>
            <p className="settings-help">
              Lists: .Tasks, .HighTasks, .MediumTasks, .LowTasks. Supports if conditions and range
              loops.
            </p>
          </div>
        )}
        <label className="field" htmlFor="discord-subject">
          <span>Message title</span>
          <input
            id="discord-subject"
            value={config.subject}
            maxLength={160}
            required
            disabled={!!busy}
            onChange={(e) => change('subject', e.target.value)}
          />
        </label>
        <label className="field" htmlFor="discord-body">
          <span>Message</span>
          <textarea
            id="discord-body"
            rows={7}
            value={config.body}
            maxLength={10000}
            required
            disabled={!!busy}
            onChange={(e) => change('body', e.target.value)}
          />
        </label>
        <p className="muted settings-help">
          Discord supports text and Markdown. The combined title and message may contain up to 2,000
          characters. Automatic mentions are disabled.
        </p>
        <div className="notify-actions">
          <button type="submit" className="btn btn-primary" disabled={!!busy}>
            {busy === 'save' ? 'Saving...' : 'Save Discord settings'}
          </button>
          <button
            type="button"
            className="btn btn-ghost"
            disabled={!!busy}
            onClick={() => action('preview')}
          >
            Preview
          </button>
          <button
            type="button"
            className="btn btn-ghost"
            disabled={!!busy || !info.configured || !!webhook}
            onClick={() => action('test')}
          >
            {busy === 'test' ? 'Sending...' : 'Send Discord test'}
          </button>
          <button
            type="button"
            className="btn btn-primary"
            disabled={!!busy || !info.configured || !!webhook}
            onClick={() => action('send')}
          >
            {busy === 'send' ? 'Sending...' : 'Send now'}
          </button>
        </div>
      </form>
      {preview && (
        <div className="notify-preview" role="status">
          <h3>Discord message preview</h3>
          <pre>{preview}</pre>
        </div>
      )}
      {success && <DiscordSuccess message={success} onClose={() => setSuccess('')} />}
    </section>
  )
}
