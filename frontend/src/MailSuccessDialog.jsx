import { useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'

export default function MailSuccessDialog({ result, onClose }) {
  const dialog = useRef(null)
  useEffect(() => {
    const element = dialog.current
    element.showModal()
    return () => element.close()
  }, [])
  return createPortal(
    <dialog
      ref={dialog}
      className="complete-dialog mail-success-dialog"
      aria-labelledby="mail-success-title"
      aria-describedby="mail-success-description"
      onCancel={(event) => {
        event.preventDefault()
        onClose()
      }}
    >
      <div className="mail-success-content">
        <span className="mail-success-icon" aria-hidden="true">
          ✓
        </span>
        <h2 id="mail-success-title">Email berhasil dikirim</h2>
        <p id="mail-success-description">
          Server SMTP Hostinger menerima email untuk diteruskan ke {result.accepted} penerima.
        </p>
        <p className="muted">Silakan periksa inbox atau folder spam penerima.</p>
        <button type="button" className="btn btn-primary" autoFocus onClick={onClose}>
          Oke
        </button>
      </div>
    </dialog>,
    document.body,
  )
}
