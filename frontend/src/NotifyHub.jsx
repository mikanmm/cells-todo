import NotifySettings from './NotifySettings.jsx'
import DiscordNotifySettings from './DiscordNotifySettings.jsx'
import Chevron from './Chevron.jsx'

export default function NotifyHub({ onSessionExpired }) {
  return (
    <div className="notify-hub">
      <div className="notify-hub-heading">
        <h2>Notify</h2>
        <p className="muted">
          Pilih saluran notifikasi untuk mengatur pesan dan jadwal pengiriman.
        </p>
      </div>
      <details className="card notify-channel">
        <summary>
          <span className="notify-channel-symbol" aria-hidden="true">
            ✉
          </span>
          <span>
            <strong>Notify By Mail</strong>
            <small>Kirim pengingat ke akun email Anda.</small>
          </span>
          <span className="notify-channel-chevron">
            <Chevron />
          </span>
        </summary>
        <NotifySettings embedded onSessionExpired={onSessionExpired} />
      </details>
      <details className="card notify-channel">
        <summary>
          <span className="notify-channel-symbol discord" aria-hidden="true">
            #
          </span>
          <span>
            <strong>Notify By Discord Webhook</strong>
            <small>Kirim pengingat ke channel Discord Anda.</small>
          </span>
          <span className="notify-channel-chevron">
            <Chevron />
          </span>
        </summary>
        <DiscordNotifySettings onSessionExpired={onSessionExpired} />
      </details>
    </div>
  )
}
