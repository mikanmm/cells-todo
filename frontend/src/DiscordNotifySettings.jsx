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
        <h2 id="discord-success-title">Pesan berhasil dikirim</h2>
        <p>{message}</p>
        <button className="btn btn-primary" type="button" autoFocus onClick={onClose}>
          Oke
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
        setNotice('Webhook, template, dan jadwal Discord berhasil disimpan.')
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
              Muat ulang
            </button>
          </>
        ) : (
          <p role="status">Memuat pengaturan Discord...</p>
        )}
      </section>
    )
  return (
    <section className="settings-panel discord-panel" aria-busy={!!busy}>
      <div>
        <h2>Discord Webhook</h2>
        <p className="muted">Kirim pengingat ke channel teks Discord melalui webhook.</p>
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
        {info.configured ? 'Webhook tersimpan' : 'Webhook belum dikonfigurasi'}
      </p>
      <form
        onSubmit={(e) => {
          e.preventDefault()
          action('save')
        }}
      >
        <label className="field" htmlFor="discord-webhook">
          <span>URL webhook Discord</span>
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
                ? 'Kosongkan untuk mempertahankan webhook tersimpan'
                : 'https://discord.com/api/webhooks/...'
            }
          />
        </label>
        <p className="muted settings-help">
          Buat melalui pengaturan channel Discord → Integrations → Webhooks. URL disimpan di server
          dan tidak ditampilkan kembali. Simpan perubahan URL sebelum mengirim tes.
        </p>
        <label className="field" htmlFor="discord-mode">
          <span>Mode pesan</span>
          <select
            id="discord-mode"
            value={config.mode}
            disabled={!!busy}
            onChange={(e) => change('mode', e.target.value)}
          >
            <option value="text">Teks biasa</option>
            <option value="template">Script custom (Go template)</option>
          </select>
        </label>
        {config.mode === 'template' && (
          <div className="notify-hint">
            <p>Gunakan parameter tugas aktif milik Anda:</p>
            <code>{'{{.Name}} {{.Date}} {{.Total}} {{.Urgent}} {{.Medium}} {{.Low}}'}</code>
            <p>
              <code>{'{{range .HighTasks}}- {{.Title}}{{end}}'}</code>
            </p>
            <p className="settings-help">
              Daftar: .Tasks, .HighTasks, .MediumTasks, .LowTasks. Mendukung kondisi if dan
              perulangan range.
            </p>
          </div>
        )}
        <label className="field" htmlFor="discord-subject">
          <span>Judul pesan</span>
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
          <span>Pesan</span>
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
          Discord mendukung teks/Markdown. Hasil judul dan pesan maksimal 2.000 karakter. Mention
          otomatis dinonaktifkan.
        </p>
        <div className="notify-schedule">
          <h3>Cronjob harian</h3>
          <label>
            <input
              type="checkbox"
              checked={config.enabled}
              disabled={!!busy}
              onChange={(e) => change('enabled', e.target.checked)}
            />{' '}
            Aktifkan pengiriman otomatis ke Discord
          </label>
          <span>Jam pengiriman (WIB)</span>
          <NotifyHourPicker
            value={config.hour}
            disabled={!!busy}
            onChange={(value) => change('hour', value)}
          />
          <p className="muted settings-help">
            Setiap hari dalam zona Asia/Jakarta. Server harus berjalan. Jadwal yang terlewat
            diproses satu kali saat server aktif kembali.
          </p>
          {info.nextRun?.Valid && (
            <p className="muted">
              Jadwal tersimpan berikutnya:{' '}
              {new Date(info.nextRun.Time).toLocaleString('id-ID', { timeZone: 'Asia/Jakarta' })}{' '}
              WIB
            </p>
          )}
          {info.lastStatus && <p className="muted">Status terakhir: {info.lastStatus}</p>}
        </div>
        <div className="notify-actions">
          <button type="submit" className="btn btn-primary" disabled={!!busy}>
            {busy === 'save' ? 'Menyimpan...' : 'Simpan pengaturan Discord'}
          </button>
          <button
            type="button"
            className="btn btn-ghost"
            disabled={!!busy}
            onClick={() => action('preview')}
          >
            Pratinjau
          </button>
          <button
            type="button"
            className="btn btn-ghost"
            disabled={!!busy || !info.configured || !!webhook}
            onClick={() => action('test')}
          >
            {busy === 'test' ? 'Mengirim...' : 'Kirim tes Discord'}
          </button>
          <button
            type="button"
            className="btn btn-primary"
            disabled={!!busy || !info.configured || !!webhook}
            onClick={() => action('send')}
          >
            {busy === 'send' ? 'Mengirim...' : 'Kirim sekarang'}
          </button>
        </div>
      </form>
      {preview && (
        <div className="notify-preview" role="status">
          <h3>Pratinjau pesan Discord</h3>
          <pre>{preview}</pre>
        </div>
      )}
      {success && <DiscordSuccess message={success} onClose={() => setSuccess('')} />}
    </section>
  )
}
