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
      setNotice('Daftar penerima berhasil disimpan.')
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
      setNotice(`${result.accepted} penerima: ${result.message}`)
      setMailSuccess(result)
    } catch (err) {
      if (err.status === 0 || err.status === 502 || err.status === 504) {
        setError(
          `${err.message} Jika koneksi terputus saat mengirim, periksa inbox atau log Hostinger sebelum mengulang pengiriman.`,
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
            ? `Template disimpan. Email dijadwalkan setiap hari pukul ${hour} WIB.`
            : 'Template disimpan. Jadwal otomatis nonaktif.',
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
        <h2>Notify</h2>
        <p className="muted">Kirim email dari akun info ke akun email Anda yang lain.</p>
      </div>
      <div className="notify-schedule">
        <h3>Cronjob harian</h3>
        <label>
          <input
            type="checkbox"
            checked={enabled}
            disabled={!!busy}
            onChange={(e) => setEnabled(e.target.checked)}
          />{' '}
          Aktifkan pengiriman otomatis
        </label>
        <div className="field">
          <span id="notify-hour-label">Jam pengiriman (WIB)</span>
          <NotifyHourPicker value={hour} onChange={setHour} disabled={!!busy} />
        </div>
        <p className="muted settings-help">
          Setiap hari pada jam yang dipilih, zona Asia/Jakarta. Tengah malam adalah 00:00. Jika jam
          hari ini sudah lewat, jadwal dimulai besok. Server harus berjalan; jadwal tertunda
          diproses saat server kembali aktif, maksimal satu pengiriman susulan.
        </p>
        {scheduleInfo?.nextRun?.Valid && (
          <p className="muted">
            Jadwal tersimpan berikutnya:{' '}
            {new Date(scheduleInfo.nextRun.Time).toLocaleString('id-ID', {
              timeZone: 'Asia/Jakarta',
            })}{' '}
            WIB
          </p>
        )}
        {scheduleInfo?.lastStatus && (
          <p className="muted" role="status">
            Status terakhir: {scheduleInfo.lastStatus}
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
            Muat ulang
          </button>
        ) : (
          <p className="muted" role="status">
            Memuat pengaturan email...
          </p>
        )
      ) : (
        <>
          <div className="notify-sender">
            <div>
              <span className="muted">Pengirim</span>
              <strong>{settings.sender}</strong>
            </div>
            <span className={`notify-status ${settings.configured ? 'ready' : ''}`}>
              {settings.configured ? 'SMTP dikonfigurasi' : 'Konfigurasi belum lengkap'}
            </span>
          </div>
          {!settings.configured && (
            <p className="notify-hint">
              {settings.configurationMessage} Setelah konfigurasi diperbarui, muat ulang halaman
              ini.
            </p>
          )}
          <form onSubmit={save}>
            <label className="field" htmlFor="notify-recipients">
              <span>Email penerima</span>
              <textarea
                id="notify-recipients"
                rows={4}
                value={recipients}
                disabled={!!busy}
                maxLength={3000}
                onChange={(e) => setRecipients(e.target.value)}
                placeholder="123@gmail.com; 234@gmail.com"
                aria-describedby="notify-recipients-help"
              />
            </label>
            <p className="muted settings-help" id="notify-recipients-help">
              Pisahkan alamat email dengan titik koma (;), contoh: 123@gmail.com; 234@gmail.com.
              Maksimal {settings.maxRecipients} penerima. Daftar ini disimpan untuk akun Anda.
            </p>
            <div className="notify-actions">
              <button type="submit" className="btn btn-primary" disabled={!!busy}>
                {busy === 'save' ? 'Menyimpan...' : 'Simpan penerima'}
              </button>
              <button
                type="button"
                className="btn btn-ghost"
                disabled={!canSend}
                onClick={() => send('test')}
              >
                {busy === 'test' ? 'Mengirim tes...' : 'Kirim email tes'}
              </button>
            </div>
            {dirty && (
              <p className="muted settings-help">
                Simpan perubahan penerima sebelum mengirim email.
              </p>
            )}
          </form>
          <form
            className="notify-compose"
            onSubmit={(e) => {
              e.preventDefault()
              send('broadcast')
            }}
          >
            <h3>Template & broadcast email</h3>
            <label className="field" htmlFor="notify-mode">
              <span>Mode pesan</span>
              <select
                id="notify-mode"
                value={mode}
                disabled={!!busy}
                onChange={(e) => {
                  setMode(e.target.value)
                  setPreview(null)
                }}
              >
                <option value="text">Teks biasa</option>
                <option value="template">Script custom (Go template)</option>
              </select>
            </label>
            {mode === 'template' && (
              <div className="notify-hint">
                <p>
                  Gunakan parameter dan logika Go template. Data mencakup tugas Anda yang belum
                  selesai; urgent berarti prioritas tinggi.
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
                  Daftar: .Tasks, .HighTasks, .MediumTasks, .LowTasks (maksimal 100 tugas per
                  daftar); hitungan mencakup seluruh tugas aktif. Field tugas: .Title, .Priority,
                  .Category, .DueDate. Mendukung if, range, dan perbandingan seperti gt. Bahasa yang
                  didukung adalah Go template.
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
                Gunakan template tabel bawaan
              </button>
            </div>
            <label className="field" htmlFor="notify-format">
              <span>Format email</span>
              <select
                id="notify-format"
                value={format}
                disabled={!!busy}
                onChange={(e) => {
                  setFormat(e.target.value)
                  setPreview(null)
                }}
              >
                <option value="text">Teks biasa</option>
                <option value="html">HTML (tabel & format)</option>
              </select>
            </label>
            {format === 'html' && (
              <p className="muted settings-help">
                Edit HTML tabel dan parameter sesuai kebutuhan. Gunakan Pratinjau untuk melihat
                hasilnya.
              </p>
            )}
            <label className="field" htmlFor="notify-subject">
              <span>Subjek</span>
              <input
                id="notify-subject"
                value={subject}
                required
                maxLength={160}
                disabled={!!busy}
                onChange={(e) => setSubject(e.target.value)}
                placeholder="Judul email Anda"
              />
            </label>

            <label className="field" htmlFor="notify-body">
              <span>Pesan</span>
              <textarea
                id="notify-body"
                rows={7}
                value={body}
                required
                maxLength={10000}
                disabled={!!busy}
                onChange={(e) => setBody(e.target.value)}
                placeholder="Tulis pesan untuk akun email Anda..."
              />
            </label>
            <p className="muted settings-help">
              Email tes dan broadcast dikirim ke semua penerima tersimpan. Alamat penerima tidak
              ditampilkan satu sama lain. Jeda antar pengiriman adalah satu menit.
            </p>
            <div className="notify-actions">
              <button
                type="button"
                className="btn btn-ghost"
                disabled={!!busy || !subject.trim() || !body.trim()}
                onClick={() => templateAction('preview')}
              >
                Pratinjau
              </button>
              <button
                type="button"
                className="btn btn-primary"
                disabled={!!busy || dirty || !subject.trim() || !body.trim()}
                onClick={() => templateAction('schedule')}
              >
                {busy === 'schedule' ? 'Menyimpan...' : 'Simpan template & jadwal'}
              </button>
              <button
                type="submit"
                className="btn btn-primary"
                disabled={!canSend || !subject.trim() || !body.trim()}
              >
                {busy === 'broadcast'
                  ? 'Mengirim...'
                  : `Kirim sekarang ke ${settings.recipients.length} penerima`}
              </button>
            </div>
            {preview && (
              <div className="notify-preview" role="status">
                <h3>{preview.subject}</h3>
                {preview.format === 'html' ? (
                  <iframe
                    title="Pratinjau tabel email"
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
