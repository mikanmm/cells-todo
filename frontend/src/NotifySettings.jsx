import { useEffect, useState } from 'react'
import { api } from './api.js'
import NotifyHourPicker from './NotifyHourPicker.jsx'
import MailSuccessDialog from './MailSuccessDialog.jsx'

export default function NotifySettings({ onSessionExpired, embedded = false }) {
  const [settings, setSettings] = useState(null)
  const [recipients, setRecipients] = useState('')
  const [savedRecipients, setSavedRecipients] = useState('')
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')
  const [mode, setMode] = useState('text')
  const [format, setFormat] = useState('text')
  const [defaultTemplate, setDefaultTemplate] = useState(null)
  const [enabled, setEnabled] = useState(false)
  const [hour, setHour] = useState('08:00')
  const [mailSuccess, setMailSuccess] = useState(null)
  const [scheduleInfo, setScheduleInfo] = useState(null)
  const [preview, setPreview] = useState(null)
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [reload, setReload] = useState(0)
  const dirty = recipients !== savedRecipients

  function fail(err) {
    if (err.status === 401) onSessionExpired()
    else setError(err.message)
  }

  useEffect(() => {
    let active = true
    setError('')
    Promise.all([api.notifySettings(), api.notifySchedule()])
      .then(([data, scheduled]) => {
        if (!active) return
        setSettings(data)
        setRecipients(data.recipients.join('; '))
        setSavedRecipients(data.recipients.join('; '))
        setScheduleInfo(scheduled)
        setSubject(scheduled.schedule.subject)
        setBody(scheduled.schedule.body)
        setMode(scheduled.schedule.mode)
        setFormat(scheduled.schedule.format || 'text')
        setDefaultTemplate(scheduled.defaultTemplate)
        setEnabled(scheduled.schedule.enabled)
        setHour(scheduled.schedule.hour)
      })
      .catch((err) => {
        if (active) fail(err)
      })
    return () => {
      active = false
    }
  }, [reload])

  async function save(e) {
    e.preventDefault()
    if (busy) return
    setBusy('save')
    setError('')
    setNotice('')
    try {
      const values = recipients
        .split(/[\n,;]+/)
        .map((value) => value.trim())
        .filter(Boolean)
      const data = await api.saveNotifySettings(values)
      const text = data.recipients.join('; ')
      setRecipients(text)
      setSavedRecipients(text)
      setSettings((current) => ({ ...current, recipients: data.recipients }))
      setNotice('Recipient list saved successfully.')
    } catch (err) {
      fail(err)
    } finally {
      setBusy('')
    }
  }

  async function send(kind) {
    if (busy || dirty) return
    setBusy(kind)
    setError('')
    setNotice('')
    try {
      const result =
        kind === 'test'
          ? await api.testNotifyEmail()
          : await api.broadcastEmail(subject.trim(), body, mode, format)
      setNotice(`${result.accepted} recipients: ${result.message}`)
      setMailSuccess(result)
    } catch (err) {
      if (err.status === 0 || err.status === 502 || err.status === 504) {
        setError(
          `${err.message} If the connection is interrupted during delivery, check your inbox or mail server logs before sending again.`,
        )
      } else fail(err)
    } finally {
      setBusy('')
    }
  }

  const canSend = settings?.configured && settings.recipients.length > 0 && !dirty && !busy

  async function templateAction(action) {
    if (busy) return
    setBusy(action)
    setError('')
    setNotice('')
    setPreview(null)
    const template = { enabled, hour, mode, format, subject: subject.trim(), body }
    try {
      if (action === 'preview') setPreview(await api.previewNotify(template))
      else {
        setScheduleInfo(await api.saveNotifySchedule(template))
        setNotice(
          enabled
            ? `Template saved. Email delivery is scheduled daily at ${hour} WIB.`
            : 'Template saved. Automatic delivery is disabled.',
        )
      }
    } catch (err) {
      fail(err)
    } finally {
      setBusy('')
    }
  }

  function exampleTemplate() {
    if (!defaultTemplate) return
    setMode(defaultTemplate.mode)
    setFormat(defaultTemplate.format)
    setSubject(defaultTemplate.subject)
    setBody(defaultTemplate.body)
    setPreview(null)
  }

  return (
    <section className={`${embedded ? '' : 'card '}settings-panel notify-panel`} aria-busy={!!busy}>
      <div>
        <h2>Email notifications</h2>
        <p className="muted">
          Send reminders from your configured mailbox to your saved recipients.
        </p>
      </div>
      <div className="notify-schedule">
        <h3>Daily schedule</h3>
        <label>
          <input
            type="checkbox"
            checked={enabled}
            disabled={!!busy}
            onChange={(e) => setEnabled(e.target.checked)}
          />{' '}
          Enable automatic delivery
        </label>
        <div className="field">
          <span id="notify-hour-label">Delivery time (WIB)</span>
          <NotifyHourPicker value={hour} onChange={setHour} disabled={!!busy} />
        </div>
        <p className="muted settings-help">
          Runs daily at the selected time in Asia/Jakarta. Midnight is 00:00. If the selected time
          has already passed today, delivery starts tomorrow. The server must remain running; at
          most one missed delivery is processed when the server becomes available again.
        </p>
        {scheduleInfo?.nextRun?.Valid && (
          <p className="muted">
            Next scheduled delivery:{' '}
            {new Date(scheduleInfo.nextRun.Time).toLocaleString('en-GB', {
              timeZone: 'Asia/Jakarta',
            })}{' '}
            WIB
          </p>
        )}
        {scheduleInfo?.lastStatus && (
          <p className="muted" role="status">
            Last delivery status: {scheduleInfo.lastStatus}
          </p>
        )}
      </div>
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
      {!settings ? (
        error ? (
          <button className="btn btn-ghost" onClick={() => setReload((value) => value + 1)}>
            Reload
          </button>
        ) : (
          <p className="muted" role="status">
            Loading email settings...
          </p>
        )
      ) : (
        <>
          <div className="notify-sender">
            <div>
              <span className="muted">Sender</span>
              <strong>{settings.sender}</strong>
            </div>
            <span className={`notify-status ${settings.configured ? 'ready' : ''}`}>
              {settings.configured ? 'SMTP configured' : 'Configuration incomplete'}
            </span>
          </div>
          {!settings.configured && (
            <p className="notify-hint">
              {settings.configurationMessage} Reload this page after updating the configuration.
            </p>
          )}
          <form onSubmit={save}>
            <label className="field" htmlFor="notify-recipients">
              <span>Recipient email addresses</span>
              <textarea
                id="notify-recipients"
                rows={4}
                value={recipients}
                disabled={!!busy}
                maxLength={3000}
                onChange={(e) => setRecipients(e.target.value)}
                placeholder="first@example.com; second@example.com"
                aria-describedby="notify-recipients-help"
              />
            </label>
            <p className="muted settings-help" id="notify-recipients-help">
              Separate email addresses with semicolons (;), for example: first@example.com;
              second@example.com. Maximum: {settings.maxRecipients} recipients. This list is saved
              for your account.
            </p>
            <div className="notify-actions">
              <button type="submit" className="btn btn-primary" disabled={!!busy}>
                {busy === 'save' ? 'Saving...' : 'Save recipients'}
              </button>
              <button
                type="button"
                className="btn btn-ghost"
                disabled={!canSend}
                onClick={() => send('test')}
              >
                {busy === 'test' ? 'Sending test...' : 'Send test email'}
              </button>
            </div>
            {dirty && (
              <p className="muted settings-help">Save recipient changes before sending email.</p>
            )}
          </form>
          <form
            className="notify-compose"
            onSubmit={(e) => {
              e.preventDefault()
              send('broadcast')
            }}
          >
            <h3>Email template and broadcast</h3>
            <label className="field" htmlFor="notify-mode">
              <span>Message mode</span>
              <select
                id="notify-mode"
                value={mode}
                disabled={!!busy}
                onChange={(e) => {
                  setMode(e.target.value)
                  setPreview(null)
                }}
              >
                <option value="text">Plain text</option>
                <option value="template">Custom template (Go template)</option>
              </select>
            </label>
            {mode === 'template' && (
              <div className="notify-hint">
                <p>
                  Use Go template fields and expressions. Data includes your incomplete tasks;
                  urgent refers to high-priority tasks.
                </p>
                <p>
                  <code>
                    {'{{.Name}} {{.Date}} {{.Total}} {{.Urgent}} {{.High}} {{.Medium}} {{.Low}}'}
                  </code>
                </p>
                <p>
                  <code>{'{{range .HighTasks}}- {{.Title}}{{end}}'}</code>
                </p>
                <p className="settings-help">
                  Lists: .Tasks, .HighTasks, .MediumTasks, .LowTasks (up to 100 tasks per list).
                  Counts include all active tasks. Task fields: .Title, .Priority, .Category,
                  .DueDate. Supports Go template if conditions, range loops, and comparisons such as
                  gt.
                </p>
              </div>
            )}
            <div>
              <button
                type="button"
                className="link-btn"
                disabled={!!busy || !defaultTemplate}
                onClick={exampleTemplate}
              >
                Use the default table template
              </button>
            </div>
            <label className="field" htmlFor="notify-format">
              <span>Email format</span>
              <select
                id="notify-format"
                value={format}
                disabled={!!busy}
                onChange={(e) => {
                  setFormat(e.target.value)
                  setPreview(null)
                }}
              >
                <option value="text">Plain text</option>
                <option value="html">HTML (tables and formatting)</option>
              </select>
            </label>
            {format === 'html' && (
              <p className="muted settings-help">
                Customize the HTML table and template fields as needed. Select Preview to view the
                result.
              </p>
            )}
            <label className="field" htmlFor="notify-subject">
              <span>Subject</span>
              <input
                id="notify-subject"
                value={subject}
                required
                maxLength={160}
                disabled={!!busy}
                onChange={(e) => setSubject(e.target.value)}
                placeholder="Email subject"
              />
            </label>

            <label className="field" htmlFor="notify-body">
              <span>Message</span>
              <textarea
                id="notify-body"
                rows={7}
                value={body}
                required
                maxLength={10000}
                disabled={!!busy}
                onChange={(e) => setBody(e.target.value)}
                placeholder="Enter your email message..."
              />
            </label>
            <p className="muted settings-help">
              Test emails and broadcasts are sent to all saved recipients. Recipient addresses are
              hidden from one another. Allow one minute between deliveries.
            </p>
            <div className="notify-actions">
              <button
                type="button"
                className="btn btn-ghost"
                disabled={!!busy || !subject.trim() || !body.trim()}
                onClick={() => templateAction('preview')}
              >
                Preview
              </button>
              <button
                type="button"
                className="btn btn-primary"
                disabled={!!busy || dirty || !subject.trim() || !body.trim()}
                onClick={() => templateAction('schedule')}
              >
                {busy === 'schedule' ? 'Saving...' : 'Save template and schedule'}
              </button>
              <button
                type="submit"
                className="btn btn-primary"
                disabled={!canSend || !subject.trim() || !body.trim()}
              >
                {busy === 'broadcast'
                  ? 'Sending...'
                  : `Send now to ${settings.recipients.length} recipients`}
              </button>
            </div>
            {preview && (
              <div className="notify-preview" role="status">
                <h3>{preview.subject}</h3>
                {preview.format === 'html' ? (
                  <iframe
                    title="Email table preview"
                    className="notify-html-preview"
                    sandbox=""
                    referrerPolicy="no-referrer"
                    srcDoc={`<!doctype html><html><head><meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'"><meta charset="utf-8"></head><body>${preview.body}</body></html>`}
                  />
                ) : (
                  <pre>{preview.body}</pre>
                )}
              </div>
            )}
          </form>
        </>
      )}
      {mailSuccess && (
        <MailSuccessDialog result={mailSuccess} onClose={() => setMailSuccess(null)} />
      )}
    </section>
  )
}
