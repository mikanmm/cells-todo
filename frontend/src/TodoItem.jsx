import { useState } from 'react'
import { dayDiff, dueLabel } from './dates.js'
import { GemCheck } from './Gem.jsx'

const PRIORITY_LABEL = { low: 'Rendah', medium: 'Sedang', high: 'Tinggi' }

export default function TodoItem({ todo, onToggle, onEdit, onDelete }) {
  const [confirming, setConfirming] = useState(false)

  const diff = todo.dueDate ? dayDiff(todo.dueDate) : null
  const overdue = !todo.done && diff !== null && diff < 0

  return (
    <li className={`card task ${todo.done ? 'is-done' : ''}`}>
      <GemCheck
        priority={todo.priority}
        done={todo.done}
        label={`${todo.title} (prioritas ${PRIORITY_LABEL[todo.priority].toLowerCase()})`}
        onToggle={() => onToggle(todo)}
      />

      <div className="task-body">
        <p className="task-title">{todo.title}</p>
        {todo.notes && <p className="task-notes">{todo.notes}</p>}
        <p className="task-meta">
          <span className={`prio prio-${todo.priority}`}>Prioritas {PRIORITY_LABEL[todo.priority].toLowerCase()}</span>
          {todo.dueDate && (
            <span className={`due ${overdue ? 'is-overdue' : ''}`}>
              <span className="visually-hidden">Tenggat: </span>
              {dueLabel(todo.dueDate)}
            </span>
          )}
          {todo.category && <span className="chip">{todo.category}</span>}
        </p>
      </div>

      <div className="task-actions">
        {confirming ? (
          <>
            <span className="confirm-text">Hapus tugas ini?</span>
            <button type="button" className="link-btn danger" onClick={() => onDelete(todo)}>
              Hapus
            </button>
            <button type="button" className="link-btn" onClick={() => setConfirming(false)}>
              Batal
            </button>
          </>
        ) : (
          <>
            <button type="button" className="btn btn-ghost" onClick={() => onToggle(todo)}>
              {todo.done ? 'Batalkan selesai' : 'Tandai selesai'}
            </button>
            <button type="button" className="link-btn" onClick={() => onEdit(todo)}>
              Ubah
            </button>
            <button type="button" className="link-btn danger" onClick={() => setConfirming(true)}>
              Hapus
            </button>
          </>
        )}
      </div>
    </li>
  )
}
