import { useEffect, useId, useRef, useState } from 'react'
import Chevron from './Chevron.jsx'

const hours = Array.from({ length: 24 }, (_, i) => `${String(i).padStart(2, '0')}:00`)

export default function NotifyHourPicker({ value, onChange, disabled }) {
  const listId = useId()
  const [open, setOpen] = useState(false)
  const root = useRef(null)
  const trigger = useRef(null)
  const list = useRef(null)

  useEffect(() => {
    if (!open) return
    const selected = list.current.querySelector('[aria-selected="true"]')
    selected?.focus({ preventScroll: true })
    if (selected) list.current.scrollTop = selected.offsetTop
    function outside(event) {
      if (!root.current?.contains(event.target)) setOpen(false)
    }
    document.addEventListener('pointerdown', outside)
    return () => document.removeEventListener('pointerdown', outside)
  }, [open])

  function close() {
    setOpen(false)
    trigger.current?.focus({ preventScroll: true })
  }

  function navigate(event) {
    if (event.key === 'Escape') {
      event.preventDefault()
      close()
      return
    }
    if (event.key === 'Tab') {
      setOpen(false)
      return
    }
    const index = hours.indexOf(event.target.dataset.hour)
    let next
    if (event.key === 'ArrowDown') next = Math.min(23, index + 1)
    if (event.key === 'ArrowUp') next = Math.max(0, index - 1)
    if (event.key === 'Home') next = 0
    if (event.key === 'End') next = 23
    if (next === undefined) return
    event.preventDefault()
    const option = list.current.children[next]
    option.focus({ preventScroll: true })
    if (option.offsetTop < list.current.scrollTop) list.current.scrollTop = option.offsetTop
    if (option.offsetTop + option.offsetHeight > list.current.scrollTop + list.current.clientHeight)
      list.current.scrollTop = option.offsetTop + option.offsetHeight - list.current.clientHeight
  }

  return (
    <div
      className="notify-hour-picker"
      ref={root}
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) setOpen(false)
      }}
    >
      <button
        ref={trigger}
        type="button"
        className="btn btn-ghost"
        aria-label={`Delivery time ${value} WIB`}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listId}
        disabled={disabled}
        onClick={() => setOpen(!open)}
        onKeyDown={(event) => {
          if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
            event.preventDefault()
            setOpen(true)
          }
        }}
      >
        <span>{value}</span>
        <Chevron />
      </button>
      {open && (
        <div
          ref={list}
          id={listId}
          className="notify-hour-options"
          role="listbox"
          aria-label="Delivery time (WIB)"
          onKeyDown={navigate}
        >
          {hours.map((hour) => (
            <button
              key={hour}
              type="button"
              role="option"
              data-hour={hour}
              aria-selected={value === hour}
              tabIndex={-1}
              onClick={() => {
                onChange(hour)
                close()
              }}
            >
              {hour}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
