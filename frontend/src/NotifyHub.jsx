import NotifySettings from './NotifySettings.jsx'
import DiscordNotifySettings from './DiscordNotifySettings.jsx'
import Chevron from './Chevron.jsx'

export default function NotifyHub({ onSessionExpired }) {
  return (
    <div className="notify-hub">
      <div className="notify-hub-heading">
        <h2>Notifications</h2>
        <p className="muted">
          Select a notification channel to configure messages and delivery schedules.
        </p>
      </div>
      <details className="card notify-channel">
        <summary>
          <span className="notify-channel-symbol" aria-hidden="true">
            ✉
          </span>
          <span>
            <strong>Email notifications</strong>
            <small>Send reminders to your email recipients.</small>
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
            <strong>Discord notifications</strong>
            <small>Send reminders to your Discord channel.</small>
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
